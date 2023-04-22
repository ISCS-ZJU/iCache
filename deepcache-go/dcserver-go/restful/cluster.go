package restful

import (
	"encoding/json"
	"main/cluster"

	"github.com/beego/beego/v2/server/web"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"
)

type clusterController struct {
	web.Controller
}

func (clu *clusterController) Get() {
	cluster.Minfolock.Lock()
	clusterinfo, _ := json.Marshal(cluster.Minfo)
	clu.Ctx.WriteString(string(clusterinfo))
	cluster.Minfolock.Unlock()
	log.Debug("[RESTCLUSTER] GET info")
}

func (clu *clusterController) Post() {
	command := string(clu.Ctx.Input.RequestBody)
	if command == "info" {
		cluster.Minfolock.Lock()
		clusterinfo, _ := json.Marshal(cluster.Minfo)
		clu.Ctx.WriteString(string(pretty.Pretty(clusterinfo)))
		cluster.Minfolock.Unlock()
	}
}
