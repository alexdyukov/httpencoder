package httpencoder

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
)

type (
	// Encoder implements writer for http.ResponseWriter body.
	Encoder interface {
		// Encode encodes http.ResponseWriter body.
		Encode(ctx context.Context, to io.Writer, from []byte) error
		// String used to be set Content-Encoding field.
		String() string
	}
	// Decoder implements reader for http.Request body.
	Decoder interface {
		// Decode decodes http.Request.Body.
		Decode(ctx context.Context, to io.Writer, from []byte) error
	}
)

const (
	defaultQuality = 1000
)

// New returns net/http middleware for auto decode http.Request
// and/or auto encode http.ResponseWriter body based on provided Encoders/Decoders.
func New(encoders map[string]Encoder, decoders map[string]Decoder) func(next http.Handler) http.Handler {
	bufferPool := &sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	return func(next http.Handler) http.Handler {
		decodedHandler := decode(bufferPool, decoders, next)

		return encode(bufferPool, encoders, decodedHandler)
	}
}

func bufferGet(bufferPool *sync.Pool) *bytes.Buffer {
	bodyBuffer, okay := bufferPool.Get().(*bytes.Buffer)
	if !okay {
		panic("httpencoder: unreachable code")
	}

	return bodyBuffer
}

func bufferPut(bufferPool *sync.Pool, buffer *bytes.Buffer) {
	buffer.Reset()
	bufferPool.Put(buffer)
}

func isAlpha(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func compactAndLow(input []byte) []byte {
	trueEnd := 0

	for curPos := 0; curPos < len(input); curPos++ {
		if input[curPos] == '\t' || input[curPos] == ' ' {
			continue
		}

		if byte('A') <= input[curPos] && input[curPos] <= byte('Z') {
			input[trueEnd] = input[curPos] - byte('A') + byte('a')
		} else {
			input[trueEnd] = input[curPos]
		}

		trueEnd++
	}

	return input[:trueEnd]
}
