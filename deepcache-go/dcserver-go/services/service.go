package services

import (
	"main/common"

	log "github.com/sirupsen/logrus"
)

var (
	DCRuntime *deepcachemodel
)

func Start() {
	multijob_init()
	is_ratio, us_ratio := 0.0, 0.0
	if common.Config.Cache_type == "isa" && common.Config.Package_design {
		is_ratio = common.Config.Cache_ratio * 0.9
		us_ratio = common.Config.Cache_ratio * 0.1
	} else {
		is_ratio = common.Config.Cache_ratio
	}

	DCRuntime = &deepcachemodel{
		Uuid:        common.Config.Uuid,
		Cache_ratio: is_ratio,
		Cache_type:  common.Config.Cache_type,
		Data_path:   common.Config.Dcserver_data_path,
		// Resample_ratio: common.Config.Resample_ratio,
		Package_design: common.Config.Package_design,
		Us_ratio:       us_ratio,
	}
	initDeepcacheModel(DCRuntime)
	log.Info("[Mod] service module success")
}
