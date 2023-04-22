package services

import (
	"errors"
	"io/ioutil"
	"path"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
)

// deepcache meta data
type deepcachemodel struct {
	// sync.RWMutex        // rwlock
	Uuid string `json:"uuid"` // UUID

	// cache and its meta-information
	Cache_ratio float64             `json:"cache_ratio"` // ratio of no. imgs which can be cached to total imgs.
	Cache_type  string              `json:"cache_type"`  // 'lru' | 'isa' | 'coordl' | 'quiver', used to set self.cache_meta
	Cache_size  int64               `json:"cache_size"`  // cache_size
	Cache_mng   cache_mng_interface `json:"cache_mng"`   // cache_mng

	// set cache_mng by string of Cache_type
	all_cache_mng_init_funcs map[string]interface{} `json:"-"` // set cache_mng by string of Cache_type

	Isa_related_metadata *isa_related_struct `json:"isa_related_metadata"`

	// package-related parameters
	Package_design bool `josn:"design2"` // whether open design 2
	// Itt            float64 `json:"itt"`        // important tar threshold
	Us_ratio   float64 `json:"us_ratio"`   // unimportant_samples ratio of cache_size
	Us_samples int64   `json:"us_samples"` // us_samples

	// training data parameters
	Data_path      string           `json:"data_path"`      // training data path
	Noimg          int64            `json:"noimg"`          // noimg
	Imgidx2clsidx  map[int64]int64  `json:"-"`              // imgidx2cls;  imgidx to classidx
	Imgidx2imgpath map[int64]string `json:"imgidx2imgpath"` // imgidx2imgpath
	Imgpath2imgidx map[string]int64 `json:"-"`              // imgpath2imgidx
	Class2idx      map[string]int64 `json:"-"`              // class2idx
	Imgpath2clsidx map[string]int64 `json:"-"`              // imgpath2clsidx

	// quiver cache
	Package_size int64 `json:"quiver package_size"` // package_size for quiver
}

func initDeepcacheModel(dc *deepcachemodel) {
	dc.build_metamap()

	// set cache size
	noimg := len(dc.Imgidx2clsidx)
	dc.Noimg = int64(noimg)
	dc.Cache_size = int64(float64(noimg) * dc.Cache_ratio)
	// dc.Cache_size = 200
	dc.Isa_related_metadata = init_isa_related_struct(dc)

	dc.Us_samples = int64(dc.Us_ratio * float64(dc.Cache_size))

	dc.all_cache_mng_init_funcs = map[string]interface{}{
		"lru":     init_lru_cache_mng,
		"lfu":     init_lfu_cache_mng,
		"isa":     init_isa_cache_mng,
		"coordl":  init_coordl_cache_mng,
		"quiver":  init_Quiver_cache_mng,
		"isa_lru": init_lru_cache_mng,
	}

	dc.Package_size = int64(dc.Cache_size / 2) // quiver package size;
	dc.Cache_mng, _ = dc.Call(dc.Cache_type, dc)
	// log.Debug("[SERVICE-DC] *** ", dc.Cache_mng.(*Isa_cache_mng).get_type(), " ***")
	log.Info("[SERVICE-DC] *** ", dc.Cache_mng.Get_type(), " ***")

	// if quiver
	if dc.Cache_type == "quiver" {
		// dc.sequential_buffer = make(map[int64][]byte, dc.Cache_size)
		go dc.Cache_mng.(*Quiver_cache_mng).Quiver_cache_update_goroutine()
		log.Info("[SERVICE-DC] Start a cache update goroutine for Quiver cache")
	}

}

func (dc *deepcachemodel) Call(funcName string, params ...interface{}) (result cache_mng_interface, err error) {
	f := reflect.ValueOf(dc.all_cache_mng_init_funcs[funcName])
	if len(params) != f.Type().NumIn() {
		err = errors.New("the number of params is out of index")
		return
	}
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	// var res []reflect.Value
	res := f.Call(in)
	result = res[0].Interface().(cache_mng_interface)
	return
}

func (dc *deepcachemodel) build_metamap() {
	// log.Debug("[SERVICE-DC] build_metamap: ", dc.Data_path)

	// class to idx
	dc.Class2idx = make(map[string]int64)
	clsidx2class := make(map[int64]string)
	entrys, err := ioutil.ReadDir(dc.Data_path)

	if err != nil {
		log.Fatal(err)
	}
	idx := 0
	for _, entry := range entrys {
		if entry.Name()[0] != '.' && entry.IsDir() {
			// log.Infoln(entry.Name())
			dc.Class2idx[entry.Name()] = int64(idx)
			clsidx2class[int64(idx)] = entry.Name()
			idx++
		}
	}

	// imgpath to classidx
	dc.Imgpath2clsidx = make(map[string]int64)
	tmp_imgpath_slice := make([]string, dc.Noimg)
	for _, target_class := range entrys {
		// class_index := dc.Class2idx[target_class]
		target_dir := path.Join(dc.Data_path, target_class.Name())
		sub_entrys, _ := ioutil.ReadDir(target_dir)
		suffix := strings.ToUpper("JPEG")
		suffix_2 := strings.ToLower("PNG")
		for _, entry := range sub_entrys {
			// log.Infoln(entry.Name())
			entry_path := path.Join(target_dir, entry.Name())
			if strings.HasSuffix(strings.ToUpper(entry_path), suffix) ||
				strings.HasSuffix(strings.ToLower(entry_path), suffix_2) {
				dc.Imgpath2clsidx[entry_path] = dc.Class2idx[target_class.Name()]
				tmp_imgpath_slice = append(tmp_imgpath_slice, entry_path)
			}
		}
	}

	// imgpath 2 imgidx
	dc.Imgpath2imgidx = make(map[string]int64)
	idx = 0
	// for imgpath := range dc.Imgpath2clsidx {
	// 	dc.Imgpath2imgidx[imgpath] = int64(idx)
	// 	idx++
	// }
	for _, imgpath := range tmp_imgpath_slice {
		dc.Imgpath2imgidx[imgpath] = int64(idx)
		idx++
	}

	// imgidx 2 classidx
	dc.Imgidx2clsidx = make(map[int64]int64)
	for imgpath, imgidx := range dc.Imgpath2imgidx {
		dc.Imgidx2clsidx[imgidx] = dc.Imgpath2clsidx[imgpath]
	}

	// imgidx 2 imgpath
	dc.Imgidx2imgpath = make(map[int64]string)
	for imgpath, imgidx := range dc.Imgpath2imgidx {
		dc.Imgidx2imgpath[imgidx] = imgpath
	}
	// log.Infoln(dc.Imgidx2imgpath[123])
}

type KV struct {
	Key   int64
	Value float64
}

// func (dc *deepcachemodel) find_minidx_from_accu(r float64, accu_lst []float64) int64 {
// 	left := int64(0)
// 	right := int64(len(accu_lst))
// 	mid := int64(0)
// 	for left <= right {
// 		mid = int64(math.Floor(float64(right-left)/2)) + left
// 		if accu_lst[mid] >= r {
// 			right = mid - 1
// 		} else {
// 			left = mid + 1
// 		}
// 	}
// 	return mid
// }
