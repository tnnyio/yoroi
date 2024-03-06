package fasthttp

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/valyala/fasthttp"
)

// FastClient is an interface that models *http.Client.
type FastClient func(*fasthttp.Request, *fasthttp.Response) error

// client wraps a URL and provides a method that implements fasthttp.Do.
type client[I, O interface{}] struct {
	do             FastClient
	req            CreateRequestFunc[I]
	dec            DecodeResponseFunc[O]
	bufferedStream bool
}

type URI struct {
	Host string
	Path string
}

// NewClient constructs a usable Client for a single remote method.
func NewClient[I, O interface{}](method string, url URI, enc EncodeRequestFunc[I], dec DecodeResponseFunc[O], options ...ClientOption[I, O]) *client[I, O] {
	c := &client[I, O]{
		do: fasthttp.Do,
		req: func(rc *fasthttp.Request, i I) (*fasthttp.Request, error) {
			if strings.Contains(url.Host, "//") {
				url.Host = strings.Split(url.Host, "//")[1]
			}
			rc.URI().SetHost(url.Host)
			rc.URI().SetPath(url.Path)
			rc.Header.SetMethod(method)
			err := enc(rc, i)
			if err != nil {
				return rc, err
			}
			return rc, nil
		},
		dec: dec,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

func (c *client[I, O]) Call(ctx context.Context, i I) (o O, err error) {
	req, err := c.req(fasthttp.AcquireRequest(), i)
	if err != nil {
		return
	}

	if c.bufferedStream {
		var i interface{} = i
		stream, ok := i.(io.Reader)
		if !ok {
			return o, fmt.Errorf("body must be of type io.Reader when using bufferedStream")
		}
		req.SetBodyStream(stream, -1)
	}

	resp := fasthttp.AcquireResponse()
	if err = c.do(req, resp); err != nil {
		return
	}
	return c.dec(resp)
}

// ClientOption sets an optional parameter for clients.
type ClientOption[I, O interface{}] func(*client[I, O])

// BufferedStream sets whether the HTTP response body is left open, allowing it
// to be read from later. Useful for transporting a file as a buffered stream.
// That body has to be drained and closed to properly end the request.
func BufferedStream[I, O interface{}](buffered bool) ClientOption[I, O] {
	return func(c *client[I, O]) { c.bufferedStream = buffered }
}
