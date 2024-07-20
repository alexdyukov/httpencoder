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

		if strings.Contains(request.Header.Get("Accept-Encoding"), "repeate") {
			responseWriter.Header().Set("Content-Encoding", "repeate")

			responseWriter.WriteHeader(returnedStatusCode)

			err = repeaterEntity.Encode(request.Context(), responseWriter, reversed)
		} else {
			responseWriter.WriteHeader(returnedStatusCode)

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
		requestEncoder                httpencoder.Encoder
		responseDecoder               httpencoder.Decoder
		upstreamHandler               http.HandlerFunc
		testName                      string
		requestContentEncodingHeader  string
		requestAcceptEncodingHeader   string
		responseContentEncodingHeader string
		responseStatusCode            int
	}{
		{
			testName:                      "vanilla request vanilla response",
			requestEncoder:                copierEntity,
			requestContentEncodingHeader:  "",
			requestAcceptEncodingHeader:   "",
			responseDecoder:               copierEntity,
			responseContentEncodingHeader: "",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request encode response",
			requestEncoder:                copierEntity,
			requestContentEncodingHeader:  "",
			requestAcceptEncodingHeader:   "repeate",
			responseDecoder:               repeaterEntity,
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "encode request vanilla response",
			requestEncoder:                repeaterEntity,
			requestContentEncodingHeader:  "repeate",
			requestAcceptEncodingHeader:   "",
			responseDecoder:               copierEntity,
			responseContentEncodingHeader: "",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "encode request decode response",
			requestEncoder:                repeaterEntity,
			requestContentEncodingHeader:  "repeate",
			requestAcceptEncodingHeader:   "repeate",
			responseDecoder:               repeaterEntity,
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "double encode request decode response capitalized",
			requestEncoder:                repeater2Entity,
			requestContentEncodingHeader:  "Repeate, Repeate ",
			requestAcceptEncodingHeader:   "Repeate,,repeate ",
			responseDecoder:               repeaterEntity,
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request complex accept encode type 1",
			requestEncoder:                copierEntity,
			requestContentEncodingHeader:  " ",
			requestAcceptEncodingHeader:   "repeate2, repeate;q=1.0, *;q=0.1",
			responseDecoder:               repeaterEntity,
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request complex accept encode type 2",
			requestEncoder:                copierEntity,
			requestContentEncodingHeader:  " ",
			requestAcceptEncodingHeader:   "repeate;q=1.0, repeate2;q=0.8, *;q=0.1",
			responseDecoder:               repeaterEntity,
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request donothing response",
			requestEncoder:                copierEntity,
			requestContentEncodingHeader:  "",
			requestAcceptEncodingHeader:   "repeate",
			responseDecoder:               repeaterEntity,
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithIfedEncoding,
		},
	}
)

func (repeater) String() string {
	return "repeater implementation"
}

func (repeater) Encode(_ context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i++ {
		for j := 0; j < 2; j++ {
			if _, err := to.Write(from[i : i+1]); err != nil {
				return fmt.Errorf("%w", err)
			}
		}
	}

	return nil
}

func (repeater) Decode(_ context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i += 2 {
		if _, err := to.Write(from[i : i+1]); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

func (repeater2) String() string {
	return "repeater2 implementation"
}

func (repeater2) Encode(_ context.Context, to io.Writer, from []byte) error {
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
	return "copier implementation"
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

	for _, test := range tests {
		test := test
		upperTest.Run(test.testName, func(t *testing.T) {
			t.Parallel()

			netHTTPHandler := compress(test.upstreamHandler)

			buffer := &bytes.Buffer{}

			err := test.requestEncoder.Encode(context.Background(), buffer, []byte(testString))
			if err != nil {
				t.Fatal("cannot Compress test body: " + err.Error())
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", buffer)
			request.Header.Set("Content-Encoding", test.requestContentEncodingHeader)
			request.Header.Set("Accept-Encoding", test.requestAcceptEncodingHeader)

			netHTTPHandler.ServeHTTP(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()

			if response.StatusCode != returnedStatusCode {
				t.Fatalf("unexpected response status code, want %d but got %d", returnedStatusCode, response.StatusCode)
			}

			if response.Header.Get("Content-Encoding") != test.responseContentEncodingHeader {
				strFormat := "invalid Content-Encoding header in response, want %s but got %s"
				t.Fatalf(strFormat, test.responseContentEncodingHeader, response.Header.Get("Content-Encoding"))
			}

			buffer.Reset()

			_, err = buffer.ReadFrom(response.Body)
			if err != nil {
				t.Fatal("cannot Read response.Body after http request: " + err.Error())
			}

			cleanedResponseBody := &bytes.Buffer{}

			err = test.responseDecoder.Decode(context.Background(), cleanedResponseBody, buffer.Bytes())
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
