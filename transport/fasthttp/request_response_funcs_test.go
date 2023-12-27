package fasthttp_test

import (
	"net/http"
	"testing"

	fasttransport "github.com/tnnyio/yoroi/transport/fasthttp"
	fh "github.com/valyala/fasthttp"
)

func TestSetHeader(t *testing.T) {
	const (
		key = "X-Foo"
		val = "12345"
	)
	var header http.Header = make(http.Header)
	ctx := &fh.RequestCtx{}
	fasttransport.SetResponseHeader(key, val)(ctx)

	ctx.Response.Header.VisitAll(func(key, value []byte) {
		header[string(key)] = []string{string(value)}
	})
	if want, have := val, header.Get(key); want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

func TestSetContentType(t *testing.T) {
	const contentType = "application/json"
	var header http.Header = make(http.Header)
	ctx := &fh.RequestCtx{}
	fasttransport.SetContentType(contentType)(ctx)
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		header[string(key)] = []string{string(value)}
	})
	if want, have := contentType, header.Get("Content-Type"); want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}
