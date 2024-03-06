package fasthttp

import (
	"encoding/json"

	fh "github.com/valyala/fasthttp"
)

// DecodeRequestFunc extracts a user-domain request object from an HTTP
// request object. It's designed to be used in HTTP servers, for server-side
// endpoints. One straightforward DecodeRequestFunc could be something that
// JSON decodes from the request body to the concrete request type.
type DecodeRequestFunc[Request interface{}] func(*fh.RequestCtx) (request Request, err error)

// EncodeRequestFunc encodes the passed request object into the HTTP request
// object. It's designed to be used in HTTP clients, for client-side
// endpoints. One straightforward EncodeRequestFunc could be something that JSON
// encodes the object directly to the request body.
type EncodeRequestFunc[Request interface{}] func(*fh.Request, Request) error

// CreateRequestFunc creates an outgoing HTTP request based on the passed
// request object. It's designed to be used in HTTP clients, for client-side
// endpoints. It's a more powerful version of EncodeRequestFunc, and can be used
// if more fine-grained control of the HTTP request is required.
type CreateRequestFunc[Request interface{}] func(*fh.Request, Request) (*fh.Request, error)

// EncodeResponseFunc encodes the passed response object to the HTTP response
// writer. It's designed to be used in HTTP servers, for server-side
// endpoints. One straightforward EncodeResponseFunc could be something that
// JSON encodes the object directly to the response body.
type EncodeResponseFunc[Response interface{}] func(*fh.RequestCtx, Response) error

// DecodeResponseFunc extracts a user-domain response object from an HTTP
// response object. It's designed to be used in HTTP clients, for client-side
// endpoints. One straightforward DecodeResponseFunc could be something that
// JSON decodes from the response body to the concrete response type.
type DecodeResponseFunc[Response interface{}] func(*fh.Response) (response Response, err error)

func DefaultDecodeJson[Request interface{}](ctx *fh.RequestCtx) (request Request, err error) {
	if err = json.Unmarshal(ctx.Request.Body(), &request); err != nil {
		return request, err
	}
	return
}
