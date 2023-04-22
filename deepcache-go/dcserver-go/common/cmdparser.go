package common

import (
	"flag"
	"fmt"
	"os"
)

var (
	flagNode       *string
	flagCluster    *string
	flagConfigPath *string
	gossipPort     *int
	restPort       *int
	restadminPort  *int
	rpcPort        *int
	v              *bool
	version        *bool
)

func Parser() {
	flagNode = flag.String("node", "", "The node address")
	flagCluster = flag.String("cluster", "", "addr of seed node in cluster")
	flagConfigPath = flag.String("config", "./conf/dcserver.yaml", "config path")
	gossipPort = flag.Int("gossipport", 0, "gossip port")
	restPort = flag.Int("restport", 0, "rest port")
	restadminPort = flag.Int("restadminport", 0, "rest admin port")
	rpcPort = flag.Int("rpcport", 0, "rpc port")

	version = flag.Bool("version", false, "output version")
	v = flag.Bool("v", false, "output version")
	flag.Parse()
	//version
	if (*version) || (*v) {
		fmt.Println("DCSERVER", "v0.0.2")
		os.Exit(0)
	}

	configure(*flagConfigPath)

	if *flagNode != "" {
		Config.Node = *flagNode
	}

	if *flagCluster != "" {
		Config.Cluster = *flagCluster
	}

	if *gossipPort != 0 {
		Config.Gossipport = *gossipPort
	}

	if *restPort != 0 {
		Config.Restport = *restPort
	}

	if *restadminPort != 0 {
		Config.Restadminport = *restadminPort
	}

	if *rpcPort != 0 {
		Config.Rpcport = *rpcPort
	}
}
