package circuitbreaker

import (
	"context"

	"github.com/sony/gobreaker"

	"github.com/tnnyio/yoroi/endpoint"
)

// Gobreaker returns an endpoint.Middleware that implements the circuit
// breaker pattern using the sony/gobreaker package. Only errors returned by
// the wrapped endpoint count against the circuit breaker's error count.
//
// See http://godoc.org/github.com/sony/gobreaker for more information.
func Gobreaker[O interface{}](cb *gobreaker.CircuitBreaker) endpoint.Middleware[O] {
	return func(next endpoint.Endpoint[O]) endpoint.Endpoint[O] {
		return func(ctx context.Context, request interface{}) (O, error) {
			var o O
			i, err := cb.Execute(func() (interface{}, error) { return next(ctx, request) })
			if err != nil {
				return o, err
			}
			{
				o, ok := i.(O)
				if !ok {
					return o, ConversionError
				}
				return o, nil
			}
		}
	}
}
