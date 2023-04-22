package hello

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// server is used to implement cacherpc.GreeterServer.
type server struct {
	UnimplementedGreeterServer
}

var (
	id   int64
	lock sync.Mutex
)

// SayHello implements
func (s *server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	lock.Lock()
	log.Printf("[Hello-SERVER] RECEIVED %v: %v", id, in.GetName())
	id++
	lock.Unlock()
	time.Sleep(time.Second)
	return &HelloReply{Message: "Hi " + in.GetName()}, nil
}

func (s *server) SayHelloAgain(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	return &HelloReply{Message: "Hello again " + in.GetName()}, nil
}

func Register(s *grpc.Server) {
	RegisterGreeterServer(s, &server{})
}
