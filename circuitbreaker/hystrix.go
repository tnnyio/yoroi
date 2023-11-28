package circuitbreaker

import (
	"context"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/tnnyio/yoroi/endpoint"
)

// Hystrix returns an endpoint.Middleware that implements the circuit
// breaker pattern using the afex/hystrix-go package.
//
// When using this circuit breaker, please configure your commands separately.
//
// See https://godoc.org/github.com/afex/hystrix-go/hystrix for more
// information.
func Hystrix[O interface{}](commandName string) endpoint.Middleware[O] {
	return func(next endpoint.Endpoint[O]) endpoint.Endpoint[O] {
		return func(ctx context.Context, request interface{}) (response O, err error) {
			if err := hystrix.Do(commandName, func() (err error) {
				response, err = next(ctx, request)
				return err
			}, nil); err != nil {
				return response, err
			}
			return response, nil
		}
	}
}
