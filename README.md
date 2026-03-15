# httpencoder - golang net/http middleware for decode requests and encode responses based on Accept-Encoding and Content-Encoding headers
[![Go Reference](https://pkg.go.dev/badge/image)](https://pkg.go.dev/github.com/alexdyukov/httpencoder)
[![Go Coverage](https://github.com/alexdyukov/httpencoder/wiki/coverage.svg)](https://raw.githack.com/wiki/alexdyukov/httpencoder/coverage.html)

## Decoding client body

According to RFCs there is no 'Accept-Encoding' header at server side response. It means you cannot tell clients (browsers, include headless browsers like curl/python's request) that your server accept any encodings. But some of the backends (for example [apache's mod_deflate](https://httpd.apache.org/docs/2.2/mod/mod_deflate.html#input)) support decoding request body, thats why the same feature exists in this package.

## Benchmarks

There is a little overhead to compare to `if strings.Contains(request.Header.Get("Accept-Encoding"), "myencoding")`:
```
$ go version && go test -bench=. -benchmem -benchtime=10000000x
go version go1.25.1 linux/amd64
goos: linux
goarch: amd64
pkg: github.com/alexdyukov/httpencoder
cpu: AMD Ryzen 7 8845H w/ Radeon 780M Graphics
BenchmarkRaw-16                         10000000               268.1 ns/op           720 B/op          5 allocs/op
BenchmarkIfedEncode-16                  10000000               640.4 ns/op          1456 B/op          9 allocs/op
BenchmarkWrappedEncodeDecode-16         10000000              1389 ns/op            1577 B/op         15 allocs/op
BenchmarkWrappedDecode-16               10000000               571.0 ns/op           752 B/op          7 allocs/op
BenchmarkWrappedEncode-16               10000000              1156 ns/op            1545 B/op         13 allocs/op
PASS
ok      github.com/alexdyukov/httpencoder       40.265s
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
or cheap version:
```
type gzipper struct{}

func (gzipper) Encode(ctx context.Context, to io.Writer, from []byte) (err error) {
	_, err := gzip.NewWriter(to).Write(from)
	if err != nil {
		return err
	}

	return gzipWriter.Flush()
}

func (gzipper) Decode(ctx context.Context, to io.Writer, from []byte) (err error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(from))
	if err != nil {
		return err
	}

	_, err = io.Copy(to, gzipReader)

	return err
}
```

## License

MIT licensed. See the included LICENSE file for details.
