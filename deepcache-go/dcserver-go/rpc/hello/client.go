package hello

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	address     = "127.0.0.1:18080"
	defaultName = "Hello-Client"
)

func Run_Client() {
	log.Info("[Hello-Client] " + address)
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("[Hello-Client] did not connect: %v", err)
	}
	defer conn.Close()
	c := NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	r, err := c.SayHello(ctx, &HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("[Hello-Client] could not greet: %v", err)
	}
	log.Printf("[Hello-Client] Greeting: %s", r.GetMessage())

	r, err = c.SayHelloAgain(ctx, &HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("[Hello-Client] could not greet: %v", err)
	}
	log.Printf("[Hello-Client] Greeting: %s", r.GetMessage())
}
