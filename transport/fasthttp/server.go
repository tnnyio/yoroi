package fasthttp

import (
	"encoding/json"
	"net/http"

	"github.com/tnnyio/log"
	"github.com/tnnyio/yoroi/endpoint"
	"github.com/tnnyio/yoroi/transport"
	fh "github.com/valyala/fasthttp"
)

// Server wraps an endpoint and implements http.Handler.
type Server[I, O interface{}] struct {
	e            endpoint.Endpoint[O]
	dec          DecodeRequestFunc[I]
	enc          EncodeResponseFunc[O]
	before       []RequestFunc
	after        []ServerResponseFunc
	errorEncoder ErrorEncoder
	finalizer    []ServerFinalizerFunc
	errorHandler transport.ErrorHandler
}

// NewServer constructs a new server, which implements http.Handler and wraps
// the provided endpoint.
func NewServer[I, O interface{}](
	e endpoint.Endpoint[O],
	dec DecodeRequestFunc[I],
	enc EncodeResponseFunc[O],
	options ...ServerOption[I, O],
) fh.RequestHandler {
	s := &Server[I, O]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: DefaultErrorEncoder,
		errorHandler: transport.NewLogErrorHandler(log.NewNopLogger()),
	}
	for _, option := range options {
		option(s)
	}
	return func(ctx *fh.RequestCtx) {
		if len(s.finalizer) > 0 {
			defer func() {
				ctx.SetUserValue(ContextKeyResponseHeaders, &ctx.Response.Header)
				ctx.SetUserValue(ContextKeyResponseSize, int64(len(ctx.Response.Body())))
				for _, f := range s.finalizer {
					f(ctx)
				}
			}()
		}

		for _, before := range s.before {
			before(ctx)
		}

		request, err := s.dec(ctx)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			s.errorEncoder(ctx, err)
			return
		}

		response, err := s.e(ctx, request)
		if err != nil {
			s.errorHandler.Handle(ctx, err)
			s.errorEncoder(ctx, err)
			return
		}

		for _, after := range s.after {
			after(ctx)
		}

		if err := s.enc(ctx, response); err != nil {
			s.errorHandler.Handle(ctx, err)
			s.errorEncoder(ctx, err)
			return
		}
	}
}

// ServerOption sets an optional parameter for servers.
type ServerOption[I, O interface{}] func(*Server[I, O])

// ServerBefore functions are executed on the HTTP request object before the
// request is decoded.
func ServerBefore[I, O interface{}](before ...RequestFunc) ServerOption[I, O] {
	return func(s *Server[I, O]) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter[I, O interface{}](after ...ServerResponseFunc) ServerOption[I, O] {
	return func(s *Server[I, O]) { s.after = append(s.after, after...) }
}

// ServerErrorEncoder is used to encode errors to the http.ResponseWriter
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting and response codes. By default,
// errors will be written with the DefaultErrorEncoder.
func ServerErrorEncoder[I, O interface{}](ee ErrorEncoder) ServerOption[I, O] {
	return func(s *Server[I, O]) { s.errorEncoder = ee }
}

// ServerErrorLogger is used to log non-terminal errors. By default, no errors
// are logged. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom ServerErrorEncoder or ServerFinalizer, both of which have access to
// the context.
// Deprecated: Use ServerErrorHandler instead.
func ServerErrorLogger[I, O interface{}](logger log.Logger) ServerOption[I, O] {
	return func(s *Server[I, O]) { s.errorHandler = transport.NewLogErrorHandler(logger) }
}

// ServerErrorHandler is used to handle non-terminal errors. By default, non-terminal errors
// are ignored. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom ServerErrorEncoder or ServerFinalizer, both of which have access to
// the context.
func ServerErrorHandler[I, O interface{}](errorHandler transport.ErrorHandler) ServerOption[I, O] {
	return func(s *Server[I, O]) { s.errorHandler = errorHandler }
}

// ServerFinalizer is executed at the end of every HTTP request.
// By default, no finalizer is registered.
func ServerFinalizer[I, O interface{}](f ...ServerFinalizerFunc) ServerOption[I, O] {
	return func(s *Server[I, O]) { s.finalizer = append(s.finalizer, f...) }
}

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx *fh.RequestCtx, err error)

// ServerFinalizerFunc can be used to perform work at the end of an HTTP
// request, after the response has been written to the client. The principal
// intended use is for request logging. In addition to the response code
// provided in the function signature, additional response parameters are
// provided in the context under keys with the ContextKeyResponse prefix.
type ServerFinalizerFunc func(ctx *fh.RequestCtx)

// NopRequestDecoder is a DecodeRequestFunc that can be used for requests that do not
// need to be decoded, and simply returns nil, nil.
func NopRequestDecoder(ctx *fh.RequestCtx) (interface{}, error) {
	return nil, nil
}

// EncodeJSONResponse is a EncodeResponseFunc that serializes the response as a
// JSON object to the ResponseWriter. Many JSON-over-HTTP services can use it as
// a sensible default. If the response implements Headerer, the provided headers
// will be applied to the response. If the response implements StatusCoder, the
// provided StatusCode will be used instead of 200.
func EncodeJSONResponse[Response interface{}](ctx *fh.RequestCtx, response Response) error {
	ctx.Response.Header.Set("Content-Type", "application/json; charset=utf-8")
	var resp interface{} = response
	if headerer, ok := resp.(Headerer); ok {
		for k, values := range headerer.Headers() {
			for _, v := range values {
				ctx.Response.Header.Add(k, v)
			}
		}
	}
	code := http.StatusOK
	if sc, ok := resp.(StatusCoder); ok {
		code = sc.StatusCode()
	}
	ctx.SetStatusCode(code)
	if code == http.StatusNoContent {
		return nil
	}
	return json.NewEncoder(ctx.Response.BodyWriter()).Encode(response)
}

// DefaultErrorEncoder writes the error to the ResponseWriter, by default a
// content type of text/plain, a body of the plain text of the error, and a
// status code of 500. If the error implements Headerer, the provided headers
// will be applied to the response. If the error implements json.Marshaler, and
// the marshaling succeeds, a content type of application/json and the JSON
// encoded form of the error will be used. If the error implements StatusCoder,
// the provided StatusCode will be used instead of 500.
func DefaultErrorEncoder(ctx *fh.RequestCtx, err error) {
	contentType, body := "text/plain; charset=utf-8", []byte(err.Error())
	if marshaler, ok := err.(json.Marshaler); ok {
		if jsonBody, marshalErr := marshaler.MarshalJSON(); marshalErr == nil {
			contentType, body = "application/json; charset=utf-8", jsonBody
		}
	}
	ctx.Response.Header.Set("Content-Type", contentType)
	if headerer, ok := err.(Headerer); ok {
		for k, values := range headerer.Headers() {
			for _, v := range values {
				ctx.Response.Header.Add(k, v)
			}
		}
	}
	code := http.StatusInternalServerError
	if sc, ok := err.(StatusCoder); ok {
		code = sc.StatusCode()
	}
	ctx.SetStatusCode(code)
	ctx.Response.SetBody(body)
}

// StatusCoder is checked by DefaultErrorEncoder. If an error value implements
// StatusCoder, the StatusCode will be used when encoding the error. By default,
// StatusInternalServerError (500) is used.
type StatusCoder interface {
	StatusCode() int
}

// Headerer is checked by DefaultErrorEncoder. If an error value implements
// Headerer, the provided headers will be applied to the response writer, after
// the Content-Type is set.
type Headerer interface {
	Headers() http.Header
}
