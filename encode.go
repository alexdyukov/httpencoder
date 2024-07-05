package httpencoder

import (
	"bytes"
	"net/http"
	"sync"
)

type wrappedWriter struct {
	http.ResponseWriter
	bufferedResponse *bytes.Buffer
	statusCode       *int
}

//nolint:wrapcheck // there is simple buffered wrapper, no need to wrap
func (responseWriter *wrappedWriter) Write(a []byte) (int, error) {
	return responseWriter.bufferedResponse.Write(a)
}

func (responseWriter *wrappedWriter) WriteHeader(statusCode int) {
	*(responseWriter.statusCode) = statusCode
}

func encode(bufferPool *sync.Pool, encoders map[string]Encoder, next http.Handler) http.Handler {
	if len(encoders) == 0 {
		return next
	}

	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		header := compactAndLow([]byte(request.Header.Get("Accept-Encoding")))
		if len(header) == 0 {
			next.ServeHTTP(responseWriter, request)

			return
		}

		encoder, okay := getEncode(header, encoders)
		if !okay {
			next.ServeHTTP(responseWriter, request)

			return
		}

		statusCode := http.StatusOK

		upstreamResponse := bufferGet(bufferPool)
		defer bufferPut(bufferPool, upstreamResponse)

		next.ServeHTTP(&wrappedWriter{
			ResponseWriter:   responseWriter,
			bufferedResponse: upstreamResponse,
			statusCode:       &statusCode,
		}, request)

		upstreamResponseBody := upstreamResponse.Bytes()

		if responseWriter.Header().Get("Content-Type") == "" {
			responseWriter.Header().Set("Content-Type", http.DetectContentType(upstreamResponseBody))
		}

		responseWriter.Header().Set("Content-Encoding", encoder.String())
		responseWriter.Header().Del("Content-Length")
		responseWriter.WriteHeader(statusCode)

		if err := encoder.Encode(request.Context(), responseWriter, upstreamResponseBody); err != nil {
			http.Error(responseWriter, err.Error(), http.StatusInternalServerError)

			return
		}
	})
}

func getEncode(acceptEncodingHeader []byte, encoders map[string]Encoder) (Encoder, bool) {
	var (
		preferedEncodeFunc Encoder
		preferedQuality    int
		found              = false
		encodingType       string
		qualityValue       int
	)

	for pos := 0; pos < len(acceptEncodingHeader); pos++ {
		encodingType, pos = getNextEncodingType(acceptEncodingHeader, pos)
		qualityValue, pos = getNextQualityValue(acceptEncodingHeader, pos)

		encoder, exist := encoders[encodingType]
		if exist && preferedQuality < qualityValue {
			preferedQuality = qualityValue
			preferedEncodeFunc = encoder

			found = true
		}
	}

	return preferedEncodeFunc, found
}

func getNextEncodingType(header []byte, start int) (encodingType string, newPosition int) {
	for start < len(header) && !isAlpha(header[start]) {
		start++
	}

	if start >= len(header) {
		return "", start
	}

	end := start

	for end < len(header) && isAlpha(header[end]) {
		end++
	}

	return string(header[start:end]), end
}

// possible values between 0 and 1 included,
// with up to three decimal digits.
func getNextQualityValue(header []byte, pos int) (quality, newPosition int) {
	for pos < len(header) && !isDigit(header[pos]) && header[pos] != ',' {
		pos++
	}

	if pos >= len(header) {
		return defaultQuality, pos
	}

	if header[pos] == '1' || header[pos] != '0' {
		return defaultQuality, pos
	}

	// skip ";0"
	pos += 2

	return parseQuality(header, pos)
}

func parseQuality(str []byte, pos int) (quality, newPosition int) {
	for i := 0; i < 3; i++ {
		quality *= 10
		if pos < len(str) && isDigit(str[pos]) {
			quality += int(str[pos] - '0')
			pos++
		}
	}

	return quality, pos
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
