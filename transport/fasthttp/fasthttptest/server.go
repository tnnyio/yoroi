package fasthttptest

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"testing"

	"github.com/valyala/fasthttp"
	fh "github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func FastServerHandler(handler fh.RequestHandler, req *http.Request) (*http.Response, error) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		err := fh.Serve(ln, handler)
		if err != nil {
			panic(fmt.Errorf("failed to serve: %v", err))
		}
	}()

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return ln.Dial()
			},
		},
	}

	return client.Do(req)
}

type FastHttpServer struct {
	net.Listener
	URL string
}

func FastServer(t *testing.T, handler fh.RequestHandler) *FastHttpServer {
	port := 3000 + rand.Intn(10000-1000)

	url := fmt.Sprintf("localhost:%d", port)
	ln, err := net.Listen("tcp", url)
	if err != nil {
		t.Fatalf("cannot start tcp server on port %d: %s", port, err)
	}
	go fasthttp.Serve(ln, handler)
	return &FastHttpServer{
		Listener: ln,
		URL:      "http://" + url,
	}
}
