package main

import (
	"main/common"
	"main/distkv"
	"main/restful"
	"main/rpc"
	"main/services"
	"runtime"

	log "github.com/sirupsen/logrus"
)

func init() {
	common.Parser()  // cmdline and config
	common.Logrus()  // logrus init
	services.Start() // service init
	restful.Start()  // rest init
	rpc.Start()      // rpc init
	distkv.Start()   // distributed kv etcd init
}

func main() {
	// cluster.New()
	log.Info("[Main] DeepCache Server running success:", common.Config.Name)
	// if services.DCRuntime.Cache_type == "quiver" {
	// 	// start a cache fullfill goroutine
	// 	go services.DCRuntime.Quiver_cache_update_goroutine()
	// 	log.Info("[Main] Start a cache update goroutine for Quiver cache")
	// }
	done := make(chan bool)
	log.Info("[Main] GOMAXPROCS: ", runtime.GOMAXPROCS(0))
	<-done
}
