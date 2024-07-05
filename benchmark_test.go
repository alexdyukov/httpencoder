package httpencoder_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexdyukov/httpencoder"
)

func BenchmarkRaw(b *testing.B) {
	b.StopTimer()

	body := bytes.NewBufferString("abcd")
	request := httptest.NewRequest(http.MethodPost, "/", body)

	handler := handlerWithoutEncoding

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
}

func BenchmarkRawEncode(b *testing.B) {
	b.StopTimer()

	body := bytes.NewBufferString("abcd")
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Accept-Encoding", "repeate")

	handler := handlerWithIfedEncoding

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
}

func BenchmarkWrappedEncodeDecode(b *testing.B) {
	b.StopTimer()

	body := bytes.NewBufferString("aabbccdd")
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Accept-Encoding", "repeate")
	request.Header.Set("Content-Encoding", "repeate")

	encoders := map[string]httpencoder.Encoder{"repeate": repeaterEntity}
	decoders := map[string]httpencoder.Decoder{"repeate": repeaterEntity}
	compress := httpencoder.New(encoders, decoders)
	handler := compress(handlerWithoutEncoding)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
}

func BenchmarkWrappedDecode(b *testing.B) {
	b.StopTimer()

	body := bytes.NewBufferString("aabbccdd")
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Content-Encoding", "repeate")

	encoders := map[string]httpencoder.Encoder{}
	decoders := map[string]httpencoder.Decoder{"repeate": repeaterEntity}
	compress := httpencoder.New(encoders, decoders)
	handler := compress(handlerWithoutEncoding)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
}

func BenchmarkWrappedEncode(b *testing.B) {
	b.StopTimer()

	body := bytes.NewBufferString("abcd")
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Accept-Encoding", "repeate")

	encoders := map[string]httpencoder.Encoder{"repeate": repeaterEntity}
	decoders := map[string]httpencoder.Decoder{}
	compress := httpencoder.New(encoders, decoders)
	handler := compress(handlerWithoutEncoding)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
}
