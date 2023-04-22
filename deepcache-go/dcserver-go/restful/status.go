package restful

import (
	"encoding/json"
	"main/common"
	"main/services"
	"path/filepath"

	"github.com/beego/beego/v2/server/web"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"
)

type statisticController struct {
	web.Controller
}

type statistic_data struct {
	// Epoch          int64 `json:"Epoch"`           // number of hit requests
	// Hit_nitems     int64 `json:"hit-samples"`     // number of hit requests
	// Request_nitems int64 `json:"request-samples"` // number of total requests
	Dcserver_data_path string  `json:"dcserver_data_path"` // data path
	Cache_ratio        float64 `json:"cache_ratio"`        // ratio of no. imgs which can be cached to total imgs.
	Cache_type         string  `json:"cache_type"`         // 'lru' | 'isa' | 'coordl' | 'quiver', used to set self.cache_meta
	Multijob_support   bool    `json:"mjob_enable"`        // tell if multiple job
}

func (s *statisticController) Get() {
	data_path := filepath.Dir(services.DCRuntime.Data_path)
	log.Debug("[REST-STAT] get path", data_path)
	stat := statistic_data{Dcserver_data_path: data_path,
		Cache_ratio:      services.DCRuntime.Cache_ratio,
		Cache_type:       services.DCRuntime.Cache_type,
		Multijob_support: common.Config.Multijob_support,
	}
	j, _ := json.Marshal(stat)
	s.Ctx.WriteString(string(pretty.Pretty(j)))
}
