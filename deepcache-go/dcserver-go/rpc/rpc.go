package rpc

import (
	"main/common"
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func Start() {
	go run_grpc_server()
}

func run_grpc_server() {
	var curaddr = ""
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal("[distkv.go] Get IP addr err" + err.Error())
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				curaddr = ipnet.IP.String()
				break
			}
		}
	}
	// lis, err := net.Listen("tcp", common.Config.Node+":"+strconv.Itoa(common.Config.Rpcport))
	lis, err := net.Listen("tcp", curaddr+":"+strconv.Itoa(common.Config.Rpcport))
	if err != nil {
		log.Fatalf("[RPC] listen rpc port failed %v", err)
	} else {
		log.Infoln("[RPC] GRPC server success") //, "grpc://"+common.Config.Node+":"+strconv.FormatInt(int64(common.Config.Rpcport), 10))
	}
	s := grpc.NewServer()
	Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("[RPC] GRPC server failed %v", err)
	}
}
