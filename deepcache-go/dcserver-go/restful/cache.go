package restful

import (
	"encoding/json"
	"main/services"

	"github.com/beego/beego/v2/server/web"
	"github.com/tidwall/pretty"
)

type cacheController struct {
	web.Controller
}

func (cache *cacheController) Get() {
	services.DCRuntime.Isa_related_metadata.RLock()
	defer services.DCRuntime.Isa_related_metadata.RUnlock()
	deepcache, _ := json.Marshal(services.DCRuntime)
	cache.Ctx.WriteString(string(pretty.Pretty(deepcache)))
}
