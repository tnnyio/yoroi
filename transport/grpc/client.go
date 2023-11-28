package grpc

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/tnnyio/yoroi/endpoint"
)

// Client wraps a gRPC connection and provides a method that implements
// endpoint.Endpoint.
type Client[I, O interface{}] struct {
	client      *grpc.ClientConn
	serviceName string
	method      string
	enc         EncodeRequestFunc[I]
	dec         DecodeResponseFunc[O]
	grpcReply   reflect.Type
	before      []ClientRequestFunc
	after       []ClientResponseFunc
	finalizer   []ClientFinalizerFunc
}

// NewClient constructs a usable Client for a single remote endpoint.
// Pass an zero-value protobuf message of the RPC response type as
// the grpcReply argument.
func NewClient[I, O interface{}](
	cc *grpc.ClientConn,
	serviceName string,
	method string,
	enc EncodeRequestFunc[I],
	dec DecodeResponseFunc[O],
	grpcReply interface{},
	options ...ClientOption[I, O],
) *Client[I, O] {
	c := &Client[I, O]{
		client: cc,
		method: fmt.Sprintf("/%s/%s", serviceName, method),
		enc:    enc,
		dec:    dec,
		// We are using reflect.Indirect here to allow both reply structs and
		// pointers to these reply structs. New consumers of the client should
		// use structs directly, while existing consumers will not break if they
		// remain to use pointers to structs.
		grpcReply: reflect.TypeOf(
			reflect.Indirect(
				reflect.ValueOf(grpcReply),
			).Interface(),
		),
		before: []ClientRequestFunc{},
		after:  []ClientResponseFunc{},
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// ClientOption sets an optional parameter for clients.
type ClientOption[I, O interface{}] func(*Client[I, O])

// ClientBefore sets the RequestFuncs that are applied to the outgoing gRPC
// request before it's invoked.
func ClientBefore[I, O interface{}](before ...ClientRequestFunc) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.before = append(c.before, before...) }
}

// ClientAfter sets the ClientResponseFuncs that are applied to the incoming
// gRPC response prior to it being decoded. This is useful for obtaining
// response metadata and adding onto the context prior to decoding.
func ClientAfter[I, O interface{}](after ...ClientResponseFunc) ClientOption[I, O] {
	return func(c *Client[I, O]) { c.after = append(c.after, after...) }
}

// ClientFinalizer is executed at the end of every gRPC request.
// By default, no finalizer is registered.
func ClientFinalizer[I, O interface{}](f ...ClientFinalizerFunc) ClientOption[I, O] {
	return func(s *Client[I, O]) { s.finalizer = append(s.finalizer, f...) }
}

// Endpoint returns a usable endpoint that will invoke the gRPC specified by the
// client.
// var
func (c Client[I, O]) Endpoint() endpoint.Endpoint[O] {
	return func(ctx context.Context, request interface{}) (response O, err error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if c.finalizer != nil {
			defer func() {
				for _, f := range c.finalizer {
					f(ctx, err)
				}
			}()
		}

		ctx = context.WithValue(ctx, ContextKeyRequestMethod, c.method)

		i, ok := request.(I)
		if !ok && request != nil {
			return response, fmt.Errorf(fmt.Sprintf("%v -- %v -- %v", request, reflect.TypeOf(request), reflect.TypeOf(i)))
			// return response, fmt.Errorf(fmt.Sprintf("%v -- %v", transport.InvalidRequest, request))
		}
		req, err := c.enc(ctx, i)
		if err != nil {
			return response, err
		}

		md := &metadata.MD{}
		for _, f := range c.before {
			ctx = f(ctx, md)
		}
		ctx = metadata.NewOutgoingContext(ctx, *md)

		var header, trailer metadata.MD
		grpcReply := reflect.New(c.grpcReply).Interface()
		if err = c.client.Invoke(
			ctx, c.method, req, grpcReply, grpc.Header(&header),
			grpc.Trailer(&trailer),
		); err != nil {
			return response, err
		}

		for _, f := range c.after {
			ctx = f(ctx, header, trailer)
		}

		{
			if response, err = c.dec(ctx, grpcReply); err != nil {
				return response, err
			}
		}
		return response, nil
	}
}

// ClientFinalizerFunc can be used to perform work at the end of a client gRPC
// request, after the response is returned. The principal
// intended use is for error logging. Additional response parameters are
// provided in the context under keys with the ContextKeyResponse prefix.
// Note: err may be nil. There maybe also no additional response parameters depending on
// when an error occurs.
type ClientFinalizerFunc func(ctx context.Context, err error)
