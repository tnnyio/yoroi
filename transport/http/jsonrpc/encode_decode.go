package jsonrpc

import (
	"encoding/json"

	"github.com/tnnyio/yoroi/endpoint"

	"context"
)

// Server-Side Codec

// EndpointCodec defines a server Endpoint and its associated codecs
type EndpointCodec[I, O interface{}] struct {
	Endpoint endpoint.Endpoint[O]
	Decode   DecodeRequestFunc[I]
	Encode   EncodeResponseFunc[O]
}

// EndpointCodecMap maps the Request.Method to the proper EndpointCodec
type EndpointCodecMap[I, O interface{}] map[string]EndpointCodec[I, O]

// DecodeRequestFunc extracts a user-domain request object from raw JSON
// It's designed to be used in JSON RPC servers, for server-side endpoints.
// One straightforward DecodeRequestFunc could be something that unmarshals
// JSON from the request body to the concrete request type.
type DecodeRequestFunc[I interface{}] func(context.Context, json.RawMessage) (request I, err error)

// EncodeResponseFunc encodes the passed response object to a JSON RPC result.
// It's designed to be used in HTTP servers, for server-side endpoints.
// One straightforward EncodeResponseFunc could be something that JSON encodes
// the object directly.
type EncodeResponseFunc[O interface{}] func(context.Context, O) (response json.RawMessage, err error)

// Client-Side Codec

// EncodeRequestFunc encodes the given request object to raw JSON.
// It's designed to be used in JSON RPC clients, for client-side
// endpoints. One straightforward EncodeResponseFunc could be something that
// JSON encodes the object directly.
type EncodeRequestFunc[I interface{}] func(context.Context, I) (request json.RawMessage, err error)

// DecodeResponseFunc extracts a user-domain response object from an JSON RPC
// response object. It's designed to be used in JSON RPC clients, for
// client-side endpoints. It is the responsibility of this function to decide
// whether interface{} error present in the JSON RPC response should be surfaced to the
// client endpoint.
type DecodeResponseFunc[O interface{}] func(context.Context, Response) (response O, err error)
