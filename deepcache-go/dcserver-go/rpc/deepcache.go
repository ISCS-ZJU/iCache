package rpc

import (
	context "context"
	"fmt"
	"main/rpc/cache"
	"os"

	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

// dcrpcserver imple OperatorServer
type dcrpcserver struct {
	cache.UnimplementedOperatorServer
}

// Op func imple
func (s *dcrpcserver) DCSubmit(ctx context.Context, requset *cache.DCRequest) (*cache.DCReply, error) {
	var reply *cache.DCReply
	switch requset.Type {
	case cache.OpType_get_cache_info:
		log.Debugln()
		log.Debug("[DCSERVER-RPC] get_cache_info ")
		reply, _ = Grpc_op_imple_get_cache_info(requset)
	case cache.OpType_readimg_byidx:
		log.Debugln()
		log.Debug("[DCSERVER-RPC] readimg_byidx ")
		reply, _ = Grpc_op_imple_readimg_byidx(requset)
	case cache.OpType_update_ivpersample:
		log.Debugln()
		log.Debug("[DCSERVER-RPC] update_ivpersample ")
		reply, _ = Grpc_op_imple_update_ivpersample(requset)
	case cache.OpType_readimg_byidx_t:
		log.Debugln()
		log.Debug("[DCSERVER-RPC] update_ivpersample_t ")
		reply, _ = Grpc_op_imple_readimg_byidx_t(requset)
	case cache.OpType_refresh_server_ivpsample:
		log.Debugln()
		log.Debug("[DCSERVER-RPC] refresh_server_ivpsample ")
		reply, _ = Grpc_op_imple_refresh_server_ivpsample(requset)
	default:
		log.Debugln()
		log.Warn("[DCSERVER-RPC] not suppoerted")
	}

	if reply == nil {
		reply = &cache.DCReply{}
	}

	if reply.Msg == "" {
		reply.Msg = "interface" + cache.OpType_name[int32(requset.Type)] + "not supported"
	}

	return reply, nil
}

func Register(s *grpc.Server) {
	log.Infoln("server start done. √")
	// path := "/home/tianshu/cwj/deep-cache-code/deepcache-go/dcserver-go/server_ready.txt"
	path := "server_ready.txt"
	_, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE, 0777)
	// _, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Printf("create file failed,err:%v", err)
		return
	}

	cache.RegisterOperatorServer(s, &dcrpcserver{})
}
