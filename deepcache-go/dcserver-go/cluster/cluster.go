package cluster

import (
	"encoding/base32"
	"encoding/json"
	"main/common"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
	log "github.com/sirupsen/logrus"
)

var Minfo = struct {
	Nodes   []string                      `json:"nodes"`
	Infomap map[string]common.Configmodel `json:"infomap"` // ip => configmodel
}{}
var Minfolock sync.Mutex

func Start() {
	u4 := uuid.New()
	if common.Config.Uuid == "" {
		common.Config.Uuid = u4.String()
	}
}

func New() error {
	conf := memberlist.DefaultLANConfig()
	data, _ := json.Marshal(common.Config)
	conf.Name = base32.StdEncoding.EncodeToString(data)
	conf.BindAddr = common.Config.Node
	conf.BindPort = common.Config.Gossipport
	conf.AdvertiseAddr = common.Config.Node
	conf.AdvertisePort = common.Config.Gossipport
	conf.LogOutput = common.LogWriter
	list, err := memberlist.Create(conf)
	if common.Portpool.Contains(common.Config.Gossipport) {
		log.Warning("[Cluster] Gossip port has been taken up" + strconv.Itoa(common.Config.Gossipport))
	} else {
		common.Portpool.Add(common.Config.Gossipport)
	}
	if err != nil {
		log.Fatal("failed to create memberlist:" + err.Error())
		return nil
	}
	cluster := common.Config.Cluster
	if cluster == "" {
		cluster = common.Config.Node
		common.Config.Cluster = cluster
	}
	// join cluster
	_, err = list.Join([]string{cluster})
	if err != nil {
		log.Fatal("failed to join" + err.Error())
		return nil
	}

	go func() {
		for {
			m := list.Members()
			Minfolock.Lock() // rebuild Minfo with lock
			Minfo.Nodes = []string{}
			Minfo.Infomap = make(map[string]common.Configmodel)
			for _, n := range m {
				nodedata := common.Configmodel{}
				jdata, _ := base32.StdEncoding.DecodeString(n.Name)
				jerr := json.Unmarshal(jdata, &nodedata)
				if jerr != nil {
					log.Info("[Member] Json convert failed.")
				}
				// reasign
				Minfo.Nodes = append(Minfo.Nodes, nodedata.Node)
				Minfo.Infomap[nodedata.Node] = nodedata
				log.Debugf("[Member] %s %s %d \n", nodedata.Name, n.Addr, n.Port)
			}
			Minfolock.Unlock()
			interval := common.Config.Clusterinfo_refresh_interval
			time.Sleep(time.Millisecond * interval)
		}
	}()
	return nil
}
