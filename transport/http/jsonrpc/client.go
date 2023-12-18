package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/tnnyio/yoroi/endpoint"
	"github.com/tnnyio/yoroi/transport"
	httpTransport "github.com/tnnyio/yoroi/transport/http"
)

// Client wraps a JSON RPC method and provides a method that implements endpoint.Endpoint.
type Client[I, O interface{}] struct {
	client httpTransport.HTTPClient

	// JSON RPC endpoint URL
	tgt *url.URL

	// JSON RPC method name.
	method string

	enc            EncodeRequestFunc
	dec            DecodeResponseFunc[O]
	before         []httpTransport.RequestFunc
	after          []httpTransport.ClientResponseFunc
	finalizer      httpTransport.ClientFinalizerFunc
	requestID      RequestIDGenerator
	bufferedStream bool
}

type clientRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// NewClient constructs a usable Client for a single remote method.
func NewClient[I, O interface{}](
	tgt *url.URL,
	method string,
	options ...ClientOption[I, O],
) *Client[I, O] {
	c := &Client[I, O]{
		client:         http.DefaultClient,
		method:         method,
		tgt:            tgt,
		enc:            DefaultRequestEncoder,
		dec:            DefaultResponseDecoder[O],
		before:         []httpTransport.RequestFunc{},
		after:          []httpTransport.ClientResponseFunc{},
		requestID:      NewAutoIncrementID(0),
		bufferedStream: false,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// DefaultRequestEncoder marshals the given request to JSON.
func DefaultRequestEncoder(_ context.Context, req interface{}) (json.RawMessage, error) {
	return json.Marshal(req)
}

// DefaultResponseDecoder unmarshals the result to interface{}, or returns an
// error, if found.
func DefaultResponseDecoder[O interface{}](_ context.Context, resp Response) (response O, err error) {
	if resp.Error != nil {
		return response, *resp.Error
	}
	err = json.Unmarshal(resp.Result, &response)
	return response, err
}

// ClientOption sets an optional parameter for clients.
type ClientOption[I, O interface{}] func(*Client[I, O])

// SetClient sets the underlying HTTP client used for requests.
// By default, http.DefaultClient is used.
func SetClient[I, O interface{}](client httpTransport.HTTPClient) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.client = client }
}

// ClientBefore sets the RequestFuncs that are applied to the outgoing HTTP
// request before it's invoked.
func ClientBefore[I, O interface{}](before ...httpTransport.RequestFunc) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.before = append(c.before, before...) }
}

// ClientAfter sets the ClientResponseFuncs applied to the server's HTTP
// response prior to it being decoded. This is useful for obtaining anything
// from the response and adding onto the context prior to decoding.
func ClientAfter[I, O interface{}](after ...httpTransport.ClientResponseFunc) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.after = append(c.after, after...) }
}

// ClientFinalizer is executed at the end of every HTTP request.
// By default, no finalizer is registered.
func ClientFinalizer[I, O interface{}](f httpTransport.ClientFinalizerFunc) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.finalizer = f }
}

// ClientRequestEncoder sets the func used to encode the request params to JSON.
// If not set, DefaultRequestEncoder is used.
func ClientRequestEncoder[I, O interface{}](enc EncodeRequestFunc) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.enc = enc }
}

// ClientResponseDecoder sets the func used to decode the response params from
// JSON. If not set, DefaultResponseDecoder is used.
func ClientResponseDecoder[I, O interface{}](dec DecodeResponseFunc[O]) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.dec = dec }
}

// RequestIDGenerator returns an ID for the request.
type RequestIDGenerator interface {
	Generate() interface{}
}

// ClientRequestIDGenerator is executed before each request to generate an ID
// for the request.
// By default, AutoIncrementRequestID is used.
func ClientRequestIDGenerator[I, O interface{}](g RequestIDGenerator) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.requestID = g }
}

// BufferedStream sets whether the Response.Body is left open, allowing it
// to be read from later. Useful for transporting a file as a buffered stream.
func BufferedStream[I, O interface{}](buffered bool) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.bufferedStream = buffered }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (c Client[I, O]) Endpoint() endpoint.Endpoint[O] {
	return func(ctx context.Context, request interface{}) (res O, err error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var (
			resp *http.Response
		)
		if c.finalizer != nil {
			defer func() {
				if resp != nil {
					ctx = context.WithValue(ctx, httpTransport.ContextKeyResponseHeaders, resp.Header)
					ctx = context.WithValue(ctx, httpTransport.ContextKeyResponseSize, resp.ContentLength)
				}
				c.finalizer(ctx, err)
			}()
		}

		ctx = context.WithValue(ctx, ContextKeyRequestMethod, c.method)

		var params json.RawMessage

		var i I
		{
			var ok bool
			if i, ok = request.(I); !ok {
				return res, transport.InvalidRequest
			}
		}
		if params, err = c.enc(ctx, i); err != nil {
			return res, err
		}
		rpcReq := clientRequest{
			JSONRPC: Version,
			Method:  c.method,
			Params:  params,
			ID:      c.requestID.Generate(),
		}

		req, err := http.NewRequest("POST", c.tgt.String(), nil)
		if err != nil {
			return res, err
		}

		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		var b bytes.Buffer
		req.Body = io.NopCloser(&b)
		err = json.NewEncoder(&b).Encode(rpcReq)
		if err != nil {
			return res, err
		}

		for _, f := range c.before {
			ctx = f(ctx, req)
		}

		resp, err = c.client.Do(req.WithContext(ctx))
		if err != nil {
			return res, err
		}

		if !c.bufferedStream {
			defer resp.Body.Close()
		}

		for _, f := range c.after {
			ctx = f(ctx, resp)
		}

		// Decode the body into an object
		var rpcRes Response
		err = json.NewDecoder(resp.Body).Decode(&rpcRes)
		if err != nil {
			return res, err
		}

		response, err := c.dec(ctx, rpcRes)
		if err != nil {
			return res, err
		}

		return response, nil
	}
}

// ClientFinalizerFunc can be used to perform work at the end of a client HTTP
// request, after the response is returned. The principal
// intended use is for error logging. Additional response parameters are
// provided in the context under keys with the ContextKeyResponse prefix.
// Note: err may be nil. There maybe also no additional response parameters
// depending on when an error occurs.
type ClientFinalizerFunc func(ctx context.Context, err error)

// autoIncrementID is a RequestIDGenerator that generates
// auto-incrementing integer IDs.
type autoIncrementID struct {
	v *uint64
}

// NewAutoIncrementID returns an auto-incrementing request ID generator,
// initialised with the given value.
func NewAutoIncrementID(init uint64) RequestIDGenerator {
	// Offset by one so that the first generated value = init.
	v := init - 1
	return &autoIncrementID{v: &v}
}

// Generate satisfies RequestIDGenerator
func (i *autoIncrementID) Generate() interface{} {
	id := atomic.AddUint64(i.v, 1)
	return id
}
