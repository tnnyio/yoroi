package fasthttp_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

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
		testbody = "testbody"
		encode   = func(fh *fasthttp.Request, _ Req) error {
			fh.SetBody([]byte(`{}`))
			return nil
		}
		decode = func(r *fasthttp.Response) (Res, error) {
			return TestResponse{String: string(r.Body())}, nil
		}
		headers        = make(chan string, 1)
		headerKey      = "X-Foo"
		headerVal      = "abcde"
		afterHeaderVal = "Abides"
		afterVal       = ""
	)

	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, errors.New("dang") },
		func(*fh.RequestCtx, interface{}) error { return nil },
	)
	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	client := fastTransport.NewClient[Req, Res](
		"GET",
		server.URL,
		encode,
		decode,
	)

	res, err := client.Call(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	var have string
	select {
	case have = <-headers:
	case <-time.After(time.Millisecond):
		t.Fatalf("timeout waiting for %s", headerKey)
	}
	// Check that Request Header was successfully received
	if want := headerVal; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// Check that Response header set from server was received in SetClientAfter
	if want, have := afterVal, afterHeaderVal; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// Check that the response was successfully decoded
	response, ok := res.(TestResponse)
	if !ok {
		t.Fatal("response should be TestResponse")
	}
	if want, have := testbody, response.String; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// Check that response body was closed
	b := make([]byte, 1)
	_, err = response.Body.Read(b)
	if err == nil {
		t.Fatal("wanted error, got none")
	}
	if doNotWant, have := io.EOF, err; doNotWant == have {
		t.Errorf("do not want %q, have %q", doNotWant, have)
	}
}
