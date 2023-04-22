package common

import "time"

type Configmodel struct {
	Name                         string        `default:"" json:"name"`
	Node                         string        `default:"127.0.0.1" json:"node"`
	Cluster                      string        `default:"" json:"-"`
	Gossipport                   int           `default:"18083" json:"gossipport"`
	Clusterinfo_refresh_interval time.Duration `default:"1000" json:"-"`
	Restport                     int           `default:"18082" json:"restport"`
	Restadminport                int           `default:"18081" json:"restadminport"`
	Rpcport                      int           `default:"18080" json:"rpcport"`
	Logpath                      string        `default:"" json:"-"`
	Uuid                         string        `default:"" json:"uuid"`
	Debug                        bool          `default:"true" json:"-"`
	Dcserver_data_path           string        `default:"/data/dcserver/ufs" json:"-"`
	Enableadmin                  bool          `default:"true" json:"-"`

	// cache
	Cache_ratio         float64 `default:"0.1" json:"-"`                   // ratio of no. imgs which can be cached to total imgs.
	Cache_type          string  `default:"lru" json:"-"`                   // cache type
	Resample_ratio      float64 `default:"0.0001" json:"-"`                //  resample ratio
	Async_building_task int64   `default:"100" json:"async_building_task"` // building channel
	Package_design      bool    `default:"true" json:"package_design"`     // whether open design 2
	Multijob_support    bool    `default:"true" json:"multijob_support"`   // whether open design 3
	Uc_size             int64   `default:"100" json:"uc_size"`             // uc_size
	Loading_cache_size  int64   `default:"300" json:"loading_cache_size"`  // loading_cache_size

	US_ratio   float64 `default:"0.1" json:"us_ratio"` // ratio of unimportant samples of all cache size
	Etcd_nodes string  `default:"" json:"etcd_nodes"`  // string of etcd cluster nodes
}
