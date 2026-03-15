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
	repeater  struct{}
	repeater2 struct{}
	copier    struct{}
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

		err = (copier{}).Encode(request.Context(), responseWriter, reversed)
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

			err = (repeater{}).Encode(request.Context(), responseWriter, reversed)
		} else {
			responseWriter.WriteHeader(returnedStatusCode)

			err = (copier{}).Encode(request.Context(), responseWriter, reversed)
		}

		if err != nil {
			http.Error(responseWriter, err.Error(), http.StatusInternalServerError)

			return
		}
	})

	RequestIDKey = 1

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
			requestEncoder:                copier{},
			requestContentEncodingHeader:  "",
			requestAcceptEncodingHeader:   "",
			responseDecoder:               copier{},
			responseContentEncodingHeader: "",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request encode response",
			requestEncoder:                copier{},
			requestContentEncodingHeader:  "",
			requestAcceptEncodingHeader:   "repeate",
			responseDecoder:               repeater{},
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "encode request vanilla response",
			requestEncoder:                repeater{},
			requestContentEncodingHeader:  "repeate",
			requestAcceptEncodingHeader:   "",
			responseDecoder:               copier{},
			responseContentEncodingHeader: "",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "encode request decode response",
			requestEncoder:                repeater{},
			requestContentEncodingHeader:  "repeate",
			requestAcceptEncodingHeader:   "repeate",
			responseDecoder:               repeater{},
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "double encode request decode response capitalized",
			requestEncoder:                repeater2{},
			requestContentEncodingHeader:  "Repeate, Repeate ",
			requestAcceptEncodingHeader:   "Repeate,,repeate ",
			responseDecoder:               repeater{},
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request complex accept encode type 1",
			requestEncoder:                copier{},
			requestContentEncodingHeader:  " ",
			requestAcceptEncodingHeader:   "repeate2, repeate;q=1.0, *;q=0.1",
			responseDecoder:               repeater{},
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request complex accept encode type 2",
			requestEncoder:                copier{},
			requestContentEncodingHeader:  " ",
			requestAcceptEncodingHeader:   "repeate;q=1.0, repeate2;q=0.8, *;q=0.1",
			responseDecoder:               repeater{},
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "vanilla request donothing response",
			requestEncoder:                copier{},
			requestContentEncodingHeader:  "",
			requestAcceptEncodingHeader:   "repeate",
			responseDecoder:               repeater{},
			responseContentEncodingHeader: "repeate",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithIfedEncoding,
		}, {
			testName:                      "vanilla request vanilla response cause not found encoder",
			requestEncoder:                copier{},
			requestContentEncodingHeader:  "",
			requestAcceptEncodingHeader:   "fake",
			responseDecoder:               copier{},
			responseContentEncodingHeader: "",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		}, {
			testName:                      "unknown encoding request vanilla response",
			requestEncoder:                copier{},
			requestContentEncodingHeader:  "fake",
			requestAcceptEncodingHeader:   "fake",
			responseDecoder:               copier{},
			responseContentEncodingHeader: "",
			responseStatusCode:            returnedStatusCode,
			upstreamHandler:               handlerWithoutEncoding,
		},
	}
)

func (repeater) String() string {
	return "repeater implementation"
}

func (repeater) Encode(_ context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i++ {
		for j := 0; j < 2; j++ {
			_, err := to.Write(from[i : i+1])
			if err != nil {
				return fmt.Errorf("%w", err)
			}
		}
	}

	return nil
}

func (repeater) Decode(_ context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i += 2 {
		_, err := to.Write(from[i : i+1])
		if err != nil {
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
			_, err := to.Write(from[i : i+1])
			if err != nil {
				return fmt.Errorf("%w", err)
			}
		}
	}

	return nil
}

func (repeater2) Decode(_ context.Context, to io.Writer, from []byte) error {
	for i := 0; i < len(from); i += 4 {
		_, err := to.Write(from[i : i+1])
		if err != nil {
			return fmt.Errorf("%w", err)
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
			_, err := to.Write(from[i : i+1])
			if err != nil {
				return fmt.Errorf("%w", err)
			}
		}
	}

	return nil
}

func (c copier) Decode(ctx context.Context, to io.Writer, from []byte) error {
	return c.Encode(ctx, to, from)
}

func TestEncodeDecode(test *testing.T) {
	test.Parallel()

	encoders := map[string]httpencoder.Encoder{
		"repeate": repeater{},
	}

	decoders := map[string]httpencoder.Decoder{
		"repeate": repeater{},
	}

	compress := httpencoder.New(encoders, decoders)

	for _, iterTest := range tests {
		iterTest := iterTest

		test.Run(iterTest.testName, func(t *testing.T) {
			t.Parallel()

			netHTTPHandler := compress(iterTest.upstreamHandler)

			buffer := &bytes.Buffer{}

			err := iterTest.requestEncoder.Encode(context.Background(), buffer, []byte(testString))
			if err != nil {
				t.Fatal("cannot Compress test body: " + err.Error())
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", buffer)
			request.Header.Set("Content-Encoding", iterTest.requestContentEncodingHeader)
			request.Header.Set("Accept-Encoding", iterTest.requestAcceptEncodingHeader)

			netHTTPHandler.ServeHTTP(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()

			if response.StatusCode != returnedStatusCode {
				t.Fatalf("unexpected response status code, want %d but got %d", returnedStatusCode, response.StatusCode)
			}

			if response.Header.Get("Content-Encoding") != iterTest.responseContentEncodingHeader {
				strFormat := "invalid Content-Encoding header in response, want %s but got %s"
				t.Fatalf(strFormat, iterTest.responseContentEncodingHeader, response.Header.Get("Content-Encoding"))
			}

			buffer.Reset()

			_, err = buffer.ReadFrom(response.Body)
			if err != nil {
				t.Fatal("cannot Read response.Body after http request: " + err.Error())
			}

			cleanedResponseBody := &bytes.Buffer{}

			err = iterTest.responseDecoder.Decode(context.Background(), cleanedResponseBody, buffer.Bytes())
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
