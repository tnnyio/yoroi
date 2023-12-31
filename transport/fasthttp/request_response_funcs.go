package fasthttp

import (
	"net/http"

	fh "github.com/valyala/fasthttp"
)

// RequestFunc may take information from an HTTP request and put it into a
// request context. In Servers, RequestFuncs are executed prior to invoking the
// endpoint. In Clients, RequestFuncs are executed after creating the request
// but prior to invoking the HTTP client.
type RequestFunc func(*fh.RequestCtx)

// ServerResponseFunc may take information from a request context and use it to
// manipulate a ResponseWriter. ServerResponseFuncs are only executed in
// servers, after invoking the endpoint but prior to writing a response.
type ServerResponseFunc func(*fh.RequestCtx)

// ClientResponseFunc may take information from an HTTP request and make the
// response available for consumption. ClientResponseFuncs are only executed in
// clients, after a request has been made, but prior to it being decoded.
type ClientResponseFunc func(*fh.RequestCtx) *fh.RequestCtx

// SetContentType returns a ServerResponseFunc that sets the Content-Type header
// to the provided value.
func SetContentType(contentType string) ServerResponseFunc {
	return SetResponseHeader("Content-Type", contentType)
}

// SetResponseHeader returns a ServerResponseFunc that sets the given header.
func SetResponseHeader(key, val string) ServerResponseFunc {
	return func(ctx *fh.RequestCtx) {
		ctx.Response.Header.Set(key, val)
	}
}

// SetRequestHeader returns a RequestFunc that sets the given header.
func SetRequestHeader(key, val string) RequestFunc {
	return func(ctx *fh.RequestCtx) {
		ctx.Response.Header.Set(key, val)
	}
}

// PopulateRequestContext is a RequestFunc that populates several values into
// the context from the HTTP request. Those values may be extracted using the
// corresponding ContextKey type in this package.
func PopulateRequestContext(ctx *fh.RequestCtx) {
	var header http.Header = http.Header{}
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		header[string(key)] = []string{string(value)}
	})
	for k, v := range map[contextKey]string{
		ContextKeyRequestMethod:          string(ctx.Method()),
		ContextKeyRequestURI:             ctx.URI().String(),
		ContextKeyRequestPath:            string(ctx.Path()),
		ContextKeyRequestProto:           string(ctx.Request.Header.Protocol()),
		ContextKeyRequestHost:            string(ctx.Request.Host()),
		ContextKeyRequestRemoteAddr:      ctx.RemoteAddr().String(),
		ContextKeyRequestXForwardedFor:   header.Get("X-Forwarded-For"),
		ContextKeyRequestXForwardedProto: header.Get("X-Forwarded-Proto"),
		ContextKeyRequestAuthorization:   header.Get("Authorization"),
		ContextKeyRequestReferer:         header.Get("Referer"),
		ContextKeyRequestUserAgent:       header.Get("User-Agent"),
		ContextKeyRequestXRequestID:      header.Get("X-Request-Id"),
		ContextKeyRequestAccept:          header.Get("Accept"),
	} {
		ctx.SetUserValue(k, v)
	}
}

type contextKey int

const (
	// ContextKeyRequestMethod is populated in the context by
	// PopulateRequestContext. Its value is r.Method.
	ContextKeyRequestMethod contextKey = iota

	// ContextKeyRequestURI is populated in the context by
	// PopulateRequestContext. Its value is r.RequestURI.
	ContextKeyRequestURI

	// ContextKeyRequestPath is populated in the context by
	// PopulateRequestContext. Its value is r.URL.Path.
	ContextKeyRequestPath

	// ContextKeyRequestProto is populated in the context by
	// PopulateRequestContext. Its value is r.Proto.
	ContextKeyRequestProto

	// ContextKeyRequestHost is populated in the context by
	// PopulateRequestContext. Its value is r.Host.
	ContextKeyRequestHost

	// ContextKeyRequestRemoteAddr is populated in the context by
	// PopulateRequestContext. Its value is r.RemoteAddr.
	ContextKeyRequestRemoteAddr

	// ContextKeyRequestXForwardedFor is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-For").
	ContextKeyRequestXForwardedFor

	// ContextKeyRequestXForwardedProto is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-Proto").
	ContextKeyRequestXForwardedProto

	// ContextKeyRequestAuthorization is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Authorization").
	ContextKeyRequestAuthorization

	// ContextKeyRequestReferer is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Referer").
	ContextKeyRequestReferer

	// ContextKeyRequestUserAgent is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("User-Agent").
	ContextKeyRequestUserAgent

	// ContextKeyRequestXRequestID is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Request-Id").
	ContextKeyRequestXRequestID

	// ContextKeyRequestAccept is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Accept").
	ContextKeyRequestAccept

	// ContextKeyResponseHeaders is populated in the context whenever a
	// ServerFinalizerFunc is specified. Its value is of type http.Header, and
	// is captured only once the entire response has been written.
	ContextKeyResponseHeaders

	// ContextKeyResponseSize is populated in the context whenever a
	// ServerFinalizerFunc is specified. Its value is of type int64.
	ContextKeyResponseSize
)
