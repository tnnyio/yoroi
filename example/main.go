// Copyright 2018 The Wire Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF interface{} KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The greeter binary simulates an event with greeters greeting guests.
package main

import (
	"log"
	"net"
	"os"

	logYoroi "github.com/tnnyio/log"
	"github.com/tnnyio/yoroi/example/src/proto"
	"google.golang.org/grpc"
)

func main() {
	var (
		hostPort = "localhost:9001"
		server   = grpc.NewServer()
		svc      = NewGreetService()
		logger   = logYoroi.NewLogfmtLogger(log.Writer())
	)

	sc, err := net.Listen("tcp", hostPort)
	if err != nil {
		logger.Log("Error", err)
		os.Exit(1)
	}
	defer server.GracefulStop()

	proto.RegisterGreetServiceServer(server, NewGreetServiceBinding(svc))
	logger.Log("Message", "stating gRPC server", "Address", hostPort)
	logger.Log(server.Serve(sc))
}
