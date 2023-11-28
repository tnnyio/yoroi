package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	test "github.com/tnnyio/yoroi/transport/grpc/_grpc_test"
	"github.com/tnnyio/yoroi/transport/grpc/_grpc_test/pb"
)

const (
	hostPort string = "localhost:8002"
)

func TestGRPCClient(t *testing.T) {
	var (
		server  = grpc.NewServer()
		service = test.NewService()
	)

	sc, err := net.Listen("tcp", hostPort)
	if err != nil {
		t.Fatalf("unable to listen: %+v", err)
	}
	defer server.GracefulStop()

	go func() {
		pb.RegisterTestServer(server, test.NewTestServiceBinding(service))
		pb.RegisterTestNilServer(server, test.NewTestNilServiceBinding(service))
		_ = server.Serve(sc)
	}()

	cc, err := grpc.Dial(hostPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("unable to Dial: %+v", err)
	}

	client := test.NewClient[*test.TestResponse](cc)

	var (
		a   = "the answer to life the universe and everything"
		b   = int64(42)
		cID = "request-1"
		ctx = test.SetCorrelationID(context.Background(), cID)
	)

	// Test
	responseCTX, v, err := client.Test(ctx, a, b)
	if err != nil {
		t.Fatalf("unable to Test: %+v", err)
	}
	if want, have := fmt.Sprintf("%s = %d", a, b), v; want != have {
		t.Fatalf("want %q, have %q", want, have)
	}

	if want, have := cID, test.GetConsumedCorrelationID(responseCTX); want != have {
		t.Fatalf("want %q, have %q", want, have)
	}

	// TestB
	cID = "request-2"
	ctx = test.SetCorrelationID(context.Background(), cID)
	responseCTX, v, err = client.TestB(ctx, a, b)
	if err != nil {
		t.Fatalf("unable to TestB: %+v", err)
	}
	if want, have := fmt.Sprintf("%d = %s", b, a), v; want != have {
		t.Fatalf("want %q, have %q", want, have)
	}

	if want, have := cID, test.GetConsumedCorrelationID(responseCTX); want != have {
		t.Fatalf("want %q, have %q", want, have)
	}

	// TestNil
	var empty []byte = []byte{}
	cID = "request-3"
	ctx = test.SetCorrelationID(context.Background(), cID)
	responseCTX, v, err = client.TestNil(ctx, empty)
	if err != nil {
		t.Fatalf("unable to TestNil: %+v", err)
	}
	if want, have := fmt.Sprintf("%v", empty), v; want != have {
		t.Fatalf("want %q, have %q", want, have)
	}
	if want, have := cID, test.GetConsumedCorrelationID(responseCTX); want != have {
		t.Fatalf("want %q, have %q", want, have)
	}
}
