package zipkin

import (
	"context"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"

	"github.com/tnnyio/yoroi/endpoint"
)

// TraceEndpoint returns an Endpoint middleware, tracing a Go kit endpoint.
// This endpoint tracer should be used in combination with a Go kit Transport
// tracing middleware or custom before and after transport functions as
// propagation of SpanContext is not provided in this middleware.
func TraceEndpoint[O interface{}](tracer *zipkin.Tracer, name string) endpoint.Middleware[O] {
	return func(next endpoint.Endpoint[O]) endpoint.Endpoint[O] {
		return func(ctx context.Context, request interface{}) (O, error) {
			var sc model.SpanContext
			if parentSpan := zipkin.SpanFromContext(ctx); parentSpan != nil {
				sc = parentSpan.Context()
			}
			sp := tracer.StartSpan(name, zipkin.Parent(sc))
			defer sp.Finish()

			ctx = zipkin.NewContext(ctx, sp)
			return next(ctx, request)
		}
	}
}
