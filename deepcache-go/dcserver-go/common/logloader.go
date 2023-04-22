package common

import (
	"io"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

var LogWriter io.Writer = os.Stdout

func Logrus() {
	// logpath
	if Config.Logpath != "" {
		file, err := os.OpenFile(Config.Logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
			LogWriter = file
		} else {
			log.Warn("will log to stderr")
		}
	} else {
		log.SetOutput(LogWriter)
	}

	//log.SetFormatter(&log.JSONFormatter{})
	//log.SetFormatter(&log.TextFormatter{})
	//log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if Config.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.Info("[Mod] log module start success")
	showdetails()
}

func showdetails() {
	log.Info("[Config] bind addr: ", Config.Node)
	if Config.Logpath != "" {
		log.Info("[Config] log path: ", Config.Logpath)
	}
	if Config.Enableadmin {
		log.Info("[Config] Rest:", "http://"+Config.Node+":"+strconv.FormatInt(int64(Config.Restadminport), 10))
	}
	log.Info("[Config] Rest: ", "http://"+Config.Node+":"+strconv.FormatInt(int64(Config.Restport), 10))
	log.Info("[Config] RPC grpc://"+Config.Node+":", Config.Rpcport)
	if Config.Cluster == "" {
		log.Info("[Config] Cluster gossip://"+Config.Node+":", Config.Gossipport)
	} else {
		log.Info("[Config] Cluster gossip://"+Config.Cluster+":", Config.Gossipport)
	}
}
