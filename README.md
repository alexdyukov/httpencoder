# httpencoder - golang net/http middleware for decode requests and encode responses based on Accept-Encoding and Content-Encoding headers
[![GoDoc](https://godoc.org/github.com/alexdyukov/httpencoder?status.svg)](https://godoc.org/github.com/alexdyukov/httpencoder)
[![Tests](https://github.com/alexdyukov/httpencoder/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/alexdyukov/httpencoder/actions/workflows/tests.yml?query=branch%3Amaster)

## Decoding client body

According to RFCs there is no 'Accept-Encoding' header at server side response. It means you cannot tell clients (browsers, include headless browsers like curl/python's request) that your server accept any encodings. But some of the backends (for example [apache's mod_deflate](https://httpd.apache.org/docs/2.2/mod/mod_deflate.html#input)) support decoding request body, thats why the same feature exists in this package.

## Benchmarks

There is a little overhead to compare to `if strings.Contains(request.Header.Get("Accept-Encoding"), "myencoding")`:
```
$ go test -bench=. -benchmem -benchtime=10000000x
warning: GOPATH set to GOROOT (/home/user/go) has no effect
goos: linux
goarch: amd64
pkg: github.com/alexdyukov/httpencoder
cpu: AMD Ryzen 7 8845H w/ Radeon 780M Graphics
BenchmarkRaw-16                         10000000               200.7 ns/op           720 B/op          5 allocs/op
BenchmarkIfedEncode-16                  10000000               489.1 ns/op          1456 B/op          9 allocs/op
BenchmarkWrappedEncodeDecode-16         10000000              1083 ns/op            1569 B/op         15 allocs/op
BenchmarkWrappedDecode-16               10000000               391.4 ns/op           752 B/op          7 allocs/op
BenchmarkWrappedEncode-16               10000000               917.2 ns/op          1537 B/op         13 allocs/op
PASS
ok      github.com/alexdyukov/httpencoder       30.844s
``` 

## Examples

Gzip encoder/decoder:
```

type gzipper struct{}

func (gzipper) Encode(ctx context.Context, to io.Writer, from []byte) (err error) {
	gzipWriter := gzip.NewWriter(to)

	if _, err := gzipWriter.Write(from); err != nil {
		reqID := ctx.Value(contextValueKey)

		slog.Info("failed to gzip response", "request_id", reqID, "error", err.Error())

		return fmt.Errorf("Internal server error occur. Your request id %v. Try again later or feel free to contact us to get detailed info", reqID)
	}

	if err := gzipWriter.Flush(); err != nil {
		reqID := ctx.Value(contextValueKey)

		slog.Info("failed to flush gzipped response", "request_id", reqID, "error", err.Error())

		return fmt.Errorf("Internal server error occur. Your request id %v. Try again later or feel free to contact us to get detailed info", reqID)
	}

	return nil
}

func (gzipper) Decode(ctx context.Context, to io.Writer, from []byte) (err error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(from))
	if err != nil {
		reqID := ctx.Value(contextValueKey)

		slog.Info("failed to initialize gzip reader", "request_id", reqID, "error", err.Error())

		return fmt.Errorf("Internal server error occur. Your request id %v. Try again later or feel free to contact us to get detailed info", reqID)
	}

	_, err = io.Copy(to, gzipReader)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		reqID := ctx.Value(contextValueKey)

		slog.Info("failed to read from gzip reader", "request_id", reqID, "error", err.Error())

		return fmt.Errorf("Internal server error occur. Your request id %v. Try again later or feel free to contact us to get detailed info", reqID)
	}

	return nil
}
```

## License

MIT licensed. See the included LICENSE file for details.
