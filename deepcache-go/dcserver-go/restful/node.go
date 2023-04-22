package restful

import (
	"encoding/json"
	"main/common"

	"github.com/beego/beego/v2/server/web"
	"github.com/tidwall/pretty"
)

type nodeController struct {
	web.Controller
}

func (node *nodeController) Get() {
	nodeconfig, _ := json.Marshal(common.Config)
	node.Ctx.WriteString(string(pretty.Pretty(nodeconfig)))
}
