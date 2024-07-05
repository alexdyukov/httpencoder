package httpencoder_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexdyukov/httpencoder"
)

type (
	repeater  string
	repeater2 string
	copier    string
)

const (
	returnedStatusCode = http.StatusAccepted
)

//nolint:gochecknoglobals // for reuse in different tests
var (
	testString             = "test string"
	handlerWithoutEncoding = http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		bodyRaw, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(responseWriter, err.Error(), http.StatusInternalServerError)

			return
		}

		reversed := reverse(bodyRaw)

		responseWriter.WriteHeader(returnedStatusCode)

		err = copierEntity.Encode(request.Context(), responseWriter, reversed)
		if err != nil {
			http.Error(responseWriter, err.Error(), http.StatusInternalServerError)

			return
		}
	})
	handlerWithIfedEncoding = http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		bodyRaw, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(responseWriter, err.Error(), http.StatusInternalServerError)

			return
		}

		reversed := reverse(bodyRaw)

		responseWriter.WriteHeader(returnedStatusCode)

		if strings.Contains(request.Header.Get("Accept-Encoding"), "repeate") {
			err = repeaterEntity.Encode(request.Context(), responseWriter, reversed)
		} else {
			err = copierEntity.Encode(request.Context(), responseWriter, reversed)
		}

		if err != nil {
			http.Error(responseWriter, err.Error(), http.StatusInternalServerError)

			return
		}
	})

	RequestIDKey    = 1
	copierEntity    = copier("")
	repeaterEntity  = repeater("")
	repeater2Entity = repeater2("")

	tests = []struct {
		encoder         httpencoder.Encoder
		decoder         httpencoder.Decoder
		name            string
		contentEncoding string
		acceptEncoding  string
		statusCode      int
	}{
		{
			name:            "vanilla request vanilla response",
			encoder:         copierEntity,
			decoder:         copierEntity,
			contentEncoding: "",
			acceptEncoding:  "",
			statusCode:      returnedStatusCode,
		}, {
			name:            "vanilla request encode response",
			encoder:         copierEntity,
			decoder:         repeaterEntity,
			contentEncoding: "",
			acceptEncoding:  "repeate",
			statusCode:      returnedStatusCode,
		}, {
			name:            "encode request vanilla response",
			encoder:         repeaterEntity,
			decoder:         copierEntity,
			contentEncoding: "repeate",
			acceptEncoding:  "",
			statusCode:      returnedStatusCode,
		}, {
			name:            "encode request decode response",
			encoder:         repeaterEntity,
			decoder:         repeaterEntity,
			contentEncoding: "repeate",
			acceptEncoding:  "repeate",
			statusCode:      returnedStatusCode,
		}, {
			name:            "double encode request decode response capitalized",
			encoder:         repeater2Entity,
			decoder:         repeaterEntity,
			contentEncoding: "Repeate, Repeate",
			acceptEncoding:  "Repeate,,repeate",
			statusCode:      returnedStatusCode,
		}, {
			name:            "vanilla request complex accept encode type 1",
			encoder:         copierEntity,
			decoder:         repeaterEntity,
			contentEncoding: "",
			acceptEncoding:  "repeate2, repeate;q=1.0, *;q=0.1",
			statusCode:      returnedStatusCode,
		}, {
			name:            "vanilla request complex accept encode type 2",
			encoder:         copierEntity,
			decoder:         repeaterEntity,
			contentEncoding: "",
			acceptEncoding:  "repeate;q=1.0, repeate2;q=0.8, *;q=0.1",
			statusCode:      returnedStatusCode,
		},
	}
)

func (repeater) String() string {
	return "repeater"
}

func (repeater) Encode(ctx context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i++ {
		for j := 0; j < 2; j++ {
			if _, err := to.Write(from[i : i+1]); err != nil {
				return fmt.Errorf("%w", err)
			}
		}
	}

	return nil
}

func (repeater) Decode(ctx context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i += 2 {
		if _, err := to.Write(from[i : i+1]); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

func (repeater2) String() string {
	return "repeater" // repeater, repeater used to be in content-Encoding field
}

func (repeater2) Encode(ctx context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i++ {
		for j := 0; j < 4; j++ {
			if _, err := to.Write(from[i : i+1]); err != nil {
				return fmt.Errorf("%w", err)
			}
		}
	}

	return nil
}

func (copier) String() string {
	return ""
}

func (copier) Encode(_ context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i++ {
		for j := 0; j < 1; j++ {
			if _, err := to.Write(from[i : i+1]); err != nil {
				return fmt.Errorf("%w", err)
			}
		}
	}

	return nil
}

func (c copier) Decode(ctx context.Context, to io.Writer, from []byte) error {
	return c.Encode(ctx, to, from)
}

func TestNew(upperTest *testing.T) {
	upperTest.Parallel()

	encoders := map[string]httpencoder.Encoder{
		"repeate": repeaterEntity,
	}

	decoders := map[string]httpencoder.Decoder{
		"repeate": repeaterEntity,
	}

	compress := httpencoder.New(encoders, decoders)

	netHTTPHandler := compress(handlerWithoutEncoding)

	for _, test := range tests {
		test := test
		upperTest.Run(test.name, func(t *testing.T) {
			t.Parallel()

			buffer := &bytes.Buffer{}

			err := test.encoder.Encode(context.Background(), buffer, []byte(testString))
			if err != nil {
				t.Fatal("cannot Compress test body: " + err.Error())
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", buffer)
			request.Header.Set("Content-Encoding", test.contentEncoding)
			request.Header.Set("Accept-Encoding", test.acceptEncoding)

			netHTTPHandler.ServeHTTP(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()

			if response.StatusCode != returnedStatusCode {
				t.Fatalf("unexpected response status code, want %d but got %d", returnedStatusCode, response.StatusCode)
			}

			if response.Header.Get("Content-Encoding") != fmt.Sprint(test.decoder) {
				strFormat := "invalid Content-Encoding header in response, want %s but got %s"
				t.Fatalf(strFormat, test.encoder.String(), response.Header.Get("Content-Encoding"))
			}

			buffer.Reset()

			_, err = buffer.ReadFrom(response.Body)
			if err != nil {
				t.Fatal("cannot Read response.Body after http request: " + err.Error())
			}

			cleanedResponseBody := &bytes.Buffer{}

			err = test.decoder.Decode(context.Background(), cleanedResponseBody, buffer.Bytes())
			if err != nil {
				t.Fatal("cannot Decompress test body with error: " + err.Error())
			}

			actual := cleanedResponseBody.Bytes()
			expected := reverse([]byte(testString))

			if !bytes.Equal(actual, expected) {
				t.Fatalf("invalid response: want '%v' but got '%v'", expected, actual)
			}
		})
	}
}

func reverse(str []byte) []byte {
	retVal := make([]byte, len(str))

	for i := 0; i < len(str); i++ {
		retVal[len(str)-i-1] = str[i]
	}

	return retVal
}
