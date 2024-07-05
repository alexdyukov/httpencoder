package httpencoder

import (
	"io"
	"net/http"
	"sync"
)

func decode(bufferPool *sync.Pool, decoders map[string]Decoder, next http.Handler) http.Handler {
	if len(decoders) == 0 {
		return next
	}

	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		header := compactAndLow([]byte(request.Header.Get("Content-Encoding")))
		if len(header) == 0 {
			next.ServeHTTP(responseWriter, request)

			return
		}

		bodyBuffer := bufferGet(bufferPool)
		defer bufferPut(bufferPool, bodyBuffer)

		if _, err := bodyBuffer.ReadFrom(request.Body); err != nil {
			http.Error(responseWriter, "failed to read http request body", http.StatusBadRequest)

			return
		}

		for iter := 0; iter < len(header); iter++ {
			start := iter

			for iter < len(header) && isAlpha(header[iter]) {
				iter++
			}

			decoder, exist := decoders[string(header[start:iter])]
			if !exist {
				http.Error(responseWriter, "unsupported Content-Encoding", http.StatusUnsupportedMediaType)

				return
			}

			content := bodyBuffer.Bytes()
			bodyBuffer.Reset()

			if err := decoder.Decode(request.Context(), bodyBuffer, content); err != nil {
				http.Error(responseWriter, err.Error(), http.StatusInternalServerError)

				return
			}
		}

		request.Body = io.NopCloser(bodyBuffer)
		request.Header.Del("Content-Encoding")

		next.ServeHTTP(responseWriter, request)
	})
}
