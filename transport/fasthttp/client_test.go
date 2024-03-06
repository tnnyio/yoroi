package fasthttp_test

import (
	"context"
	"io"
	"testing"

	fastTransport "github.com/tnnyio/yoroi/transport/fasthttp"
	"github.com/tnnyio/yoroi/transport/fasthttp/fasthttptest"
	"github.com/valyala/fasthttp"
	fh "github.com/valyala/fasthttp"
)

type TestResponse struct {
	Body   io.ReadCloser
	String string
}

type (
	Req interface{}
	Res interface{}
)

func TestFastHttpClient(t *testing.T) {
	var (
		encode = func(fh *fasthttp.Request, _ Req) error {
			fh.SetBody([]byte(`{}`))
			return nil
		}
		decode = func(r *fasthttp.Response) (Res, error) {
			return TestResponse{String: string(r.Body())}, nil
		}
	)

	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) {
			return struct{}{}, nil
		},
		func(req *fh.RequestCtx) (interface{}, error) {
			return struct{}{}, nil
		},
		func(*fh.RequestCtx, interface{}) error { return nil },
	)
	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	client := fastTransport.NewClient[Req, Res](
		"GET",
		fastTransport.URI{
			Host: server.URL,
		},
		encode,
		decode,
	)

	_, err := client.Call(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

}
