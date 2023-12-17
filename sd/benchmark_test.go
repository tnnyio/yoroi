package sd

import (
	"io"
	"testing"

	"github.com/tnnyio/log"
	"github.com/tnnyio/yoroi/endpoint"
)

func BenchmarkEndpoints(b *testing.B) {
	var (
		ca      = make(closer)
		cb      = make(closer)
		cmap    = map[string]io.Closer{"a": ca, "b": cb}
		factory = func(instance string) (endpoint.Endpoint[any], io.Closer, error) {
			return endpoint.Nop, cmap[instance], nil
		}
		c = newEndpointCache(factory, log.NewNopLogger(), endpointerOptions{})
	)

	b.ReportAllocs()

	c.Update(Event{Instances: []string{"a", "b"}})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Endpoints()
		}
	})
}
