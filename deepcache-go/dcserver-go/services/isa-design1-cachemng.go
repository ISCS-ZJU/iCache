package services

import (
	"container/heap"
	"context"
	"errors"
	"main/common"
	"main/distkv"
	"main/io"
	"main/rpc/cache"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type cacheEntry struct {
	imgcontent []byte // pointer to imgcontent, so that only one copy of data is cached
}

type task_type int

const (
	By_Map task_type = iota
	By_Keyvalue_update
	By_keyvalue_delete_and_update
	SWITCH_HEAP
)

type task struct {
	op_type   task_type // "by_map"
	updatemap map[int64]float64
	oldimgidx int64
	imgidx    int64
	impvalue  float64
}

// the real cache in user space
// key: imgidx; value: pointer to Node
type Isa_cache_mng struct {
	sync.RWMutex

	Cache_size   int64 `json:"cache_size"`   // cache size
	Full         bool  `json:"full"`         // cache is full
	Cache_nitems int64 `json:"cache_nitems"` // cache n items

	cache map[int64]cacheEntry // hash map to find an element quickly

	Imgidx2mainimportance   *map[int64]float64 `json:"-"` // cur using old iv
	Imgidx2shadowimportance *map[int64]float64 `json:"-"` // new iv for next epoch

	mainheap   *indexheap // main index heap
	shadowheap *indexheap // shadow index heap

	building_chanel          chan task
	Switching_shadow_channel chan int `json:"-"`
	// building_shadow_mutex sync.Mutex

	// Pack_metadata for design 2
	UC_size            int64 `json:"package_size"` // Block_size
	Loading_cache_size int64 `json:"loading_cache_size"`
	Pack_metadata      *packaging_struct

	Remote_hit int64 `json:"remote_hit_times"`
}

func init_isa_cache_mng(dc *deepcachemodel) *Isa_cache_mng {
	var isa Isa_cache_mng
	isa.Cache_size = dc.Cache_size
	isa.Full = false
	isa.Cache_nitems = 0

	isa.cache = make(map[int64]cacheEntry)

	// initialization
	tm := make(map[int64]float64)
	tm2 := make(map[int64]float64)
	isa.Imgidx2mainimportance = &tm
	isa.Imgidx2shadowimportance = &tm2

	// init main heaps
	isa.mainheap = &indexheap{Imgidx2importance: isa.Imgidx2mainimportance}
	heap.Init(isa.mainheap)

	isa.shadowheap = &indexheap{Imgidx2importance: isa.Imgidx2shadowimportance}
	heap.Init(isa.shadowheap)

	// isa.building_shadow = false
	isa.Switching_shadow_channel = make(chan int)
	isa.building_chanel = make(chan task, common.Config.Async_building_task) // DCRuntime.Resample_interval)
	go isa.async_build_shadow_heap_worker()

	// design 2
	isa.UC_size = int64(dc.Us_ratio * float64(dc.Noimg))
	log.Infof("UC_size: %v", isa.UC_size)
	isa.Loading_cache_size = common.Config.Loading_cache_size
	isa.Pack_metadata = init_packaging_struct(&isa)
	// isa.block_chan = make(chan int, 1)
	// isa.sequential_buffer = make(map[int64]*seqnode)
	// isa.accessed_n = 0
	// isa.Block_size = 100 // wj: how to decide Block_size?
	// isa.npackages = int64(math.Ceil(float64(DCRuntime.Noimg) / float64(isa.Block_size)))
	// isa.blockIDs = make(map[int64][]int64)
	// for i := int64(0); i < isa.npackages; i++ {
	// 	isa.blockIDs[i] = make([]int64, 1)
	// }
	if common.Config.Package_design {
		// pack datasets into blocks according to Block_size, write to disk
		// isa.Pack_metadata.write_packs_to_disk()
		// start a go routine to keep one block staying in sequential_buffer
		go isa.Pack_metadata.loading_thread()
		go isa.Pack_metadata.uc_thead()
	}
	log.Debugf("[CACHE ISA] init_isa_mng has complete")
	return &isa
}

func (isa *Isa_cache_mng) Get_type() string {
	return "isa"
}

func (isa *Isa_cache_mng) Exists(imgidx int64) bool {
	isa.RLock()
	defer isa.RUnlock()

	if len(isa.cache) != int(isa.Cache_size) {
		log.Errorf("isa.cache len if not[%v], but [%v]", int(isa.Cache_size), len(isa.cache))
	}

	_, ok := isa.cache[imgidx]
	if ok {
		return true
	} else {
		return false
	}
}

func (isa *Isa_cache_mng) Access(imgidx int64) ([]byte, error) {

	isa.RLock()
	defer isa.RUnlock()

	cache_entry, exist := isa.cache[imgidx]
	if exist && (cache_entry.imgcontent != nil) {
		return cache_entry.imgcontent, nil
	} else {
		return nil, errors.New("not exist")
	}
}

func (isa *Isa_cache_mng) Insert(imgidx int64, imgcontent []byte) error {
	DCRuntime.Isa_related_metadata.RLock()
	importance := DCRuntime.Isa_related_metadata.Ivpersample[imgidx]
	DCRuntime.Isa_related_metadata.RUnlock()
	// log.Info("release isa_related_metadata rlock")
	isa.Lock()
	// log.Info("got lock")
	defer isa.Unlock()

	if !isa.Full {
		log.Debugf("[ISA_CACHE_MNG] cache is NOT full")
		// push data to main_help
		(*isa.mainheap.Imgidx2importance)[imgidx] = importance
		isa.cache[imgidx] = cacheEntry{imgcontent: imgcontent}
		heap.Push(isa.mainheap, heapNode{Imgidx: imgidx})

		if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
			log.Fatal("[isa-design1-cachemng.go]", err.Error())
		} else {
			log.Infof("[isa-design1-cachemng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
		}

		isa.Sent_async_build_shadow_heap_task(By_Keyvalue_update, nil, 0, imgidx, importance)

		log.Debugf("[ISA_CACHE_MNG] push new heapNodes into heaps")

		// update meta-data
		isa.Cache_nitems += 1
		isa.Full = (isa.Cache_nitems >= isa.Cache_size)
	} else {
		log.Debugf("[ISA_CACHE_MNG] cache is full")
		// compare with first element whose importance is the minimum to decide
		// whether to cache into mainHeap or not. (*h.Imgidx2importance)[h.nodes[j].Imgidx]
		if importance > (*(*isa.mainheap).Imgidx2importance)[(*isa.mainheap).nodes[0].Imgidx] {
			log.Debug("[ISA_CACHE_MNG] insert into heap")
			// pop min node from mainHeap and insert new node to mainHeap
			tmpheapNode := heap.Pop(isa.mainheap).(heapNode)
			delete(isa.cache, tmpheapNode.Imgidx) // GC take care of heapNode in mainHeap
			delete(*isa.Imgidx2mainimportance, tmpheapNode.Imgidx)

			if err := distkv.GlobalKv.Del(tmpheapNode.Imgidx); err != nil {
				log.Fatal("[isa-design1-cachemng.go]", err.Error())
			} else {
				log.Infof("['isa-design1-cachemng.go] deleted %v from server node %v", tmpheapNode.Imgidx, distkv.Curaddr)
			}

			(*isa.mainheap.Imgidx2importance)[imgidx] = importance
			isa.cache[imgidx] = cacheEntry{imgcontent: imgcontent}
			heap.Push(isa.mainheap, heapNode{Imgidx: imgidx})

			if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
				log.Fatal("[isa-design1-cachemng.go]", err.Error())
			} else {
				log.Infof("[isa-design1-cachemng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
			}

			isa.Sent_async_build_shadow_heap_task(By_keyvalue_delete_and_update, nil, tmpheapNode.Imgidx, imgidx, importance)
		} else {
			log.Debug("[ISA_CACHE_MNG] DO NOT insert into heap")
		}
	}
	return nil
}

func (isa *Isa_cache_mng) async_build_shadow_heap_worker() {
	for build_task := range isa.building_chanel {
		log.Debug("[ISA_CACHE_MNG] async rebuilding shadow heap: ", build_task)
		// isa.Lock()
		// isa.building_shadow_mutex.Lock()
		// isa.building_shadow = true

		switch build_task.op_type {
		case By_Map:
		case By_Keyvalue_update:
			// add new value to Imgidx2shadowimportance
			(*isa.Imgidx2shadowimportance)[build_task.imgidx] = build_task.impvalue
		case By_keyvalue_delete_and_update:
			// add new value to Imgidx2shadowimportance
			delete(*isa.Imgidx2shadowimportance, build_task.oldimgidx)
			(*isa.Imgidx2shadowimportance)[build_task.imgidx] = build_task.impvalue
		case SWITCH_HEAP:
			// rebuild
			log.Debug("[ISA_CACHE_MNG] rebuild")
			for k := range *isa.Imgidx2shadowimportance {
				(*isa.Imgidx2shadowimportance)[k] = DCRuntime.Isa_related_metadata.New_Ivpersample[k]
			}
			shadowheap := &indexheap{Imgidx2importance: isa.Imgidx2shadowimportance}
			heap.Init(shadowheap)

			for imgidx := range *isa.Imgidx2shadowimportance {
				heap.Push(shadowheap, heapNode{Imgidx: imgidx})
			}
			isa.shadowheap = shadowheap
			log.Warn("[ISA_CACHE_MNG-ASYNC] switch heap!")
			isa.Switch_to_ShadowHeap()
			isa.Switching_shadow_channel <- 0
			continue
		default:
			log.Error("[ISA_CACHE_MNG] rebuilding shadow heap - Operation does not exist", build_task)
		}
		// isa.Lock()
		// shadowheap := &indexheap{Imgidx2importance: isa.Imgidx2shadowimportance}
		// heap.Init(shadowheap)

		// for imgidx := range *isa.Imgidx2shadowimportance {
		// 	heap.Push(shadowheap, heapNode{Imgidx: imgidx})
		// }

		// isa.shadowheap = shadowheap
		// isa.building_shadow = false
		// isa.building_shadow_mutex.Unlock()
		// isa.Unlock()
	}
}

func (isa *Isa_cache_mng) Sent_async_build_shadow_heap_task(op_type task_type, updatemap map[int64]float64, oldimgidx int64, imgidx int64, impvalue float64) {
	build_task := task{op_type: op_type, updatemap: updatemap, oldimgidx: oldimgidx, imgidx: imgidx, impvalue: impvalue}
	isa.building_chanel <- build_task
}

func (isa *Isa_cache_mng) Switch_to_ShadowHeap() error {
	// isa.Lock()
	// defer isa.Unlock()
	// isa.building_shadow_mutex.Lock()
	// if !isa.building_shadow {
	for k := range *isa.Imgidx2shadowimportance {
		_, ok := isa.cache[k]
		if !ok {
			log.Infof("[ISA_CACHE_MNG] debug%v %v", k, ok)
		}
	}
	tidx2importance := make(map[int64]float64)
	for idx, impval := range *isa.Imgidx2shadowimportance {
		tidx2importance[idx] = impval
	}
	isa.Imgidx2mainimportance = &tidx2importance
	isa.mainheap.nodes = isa.shadowheap.nodes
	isa.mainheap.Imgidx2importance = isa.Imgidx2mainimportance
	// } else {
	// 	log.Error("[ISA_CACHE_MNG] An error occurred during switch heap")
	// }

	// isa.building_shadow_mutex.Unlock()
	return nil
}

func (isa *Isa_cache_mng) Is_Full() bool {

	isa.RLock()
	defer isa.RUnlock()
	return isa.Full
}

func (isa *Isa_cache_mng) Get_min_heap_iv() float64 {
	isa.RLock()
	defer isa.RUnlock()
	return (*(*isa.mainheap).Imgidx2importance)[(*isa.mainheap).nodes[0].Imgidx]
}

func (isa *Isa_cache_mng) AccessAtOnce(imgidx int64) ([]byte, int64, bool) {

	isa.RLock()

	cache_entry, ok := isa.cache[imgidx]
	isa.RUnlock()
	if ok {
		// start := time.Now()
		imgcontent := cache_entry.imgcontent
		// elapsed := time.Since(start)
		// log.Printf("time read a file from local-cache [%v]:", elapsed)
		return imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[isa-design1-cachemng.go] ip:", ip)
			// start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx_t, Imgidx: imgidx, FullAccess: false})
			if err != nil {
				log.Info("[isa-design1-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
				imgpath := DCRuntime.Imgidx2imgpath[imgidx]
				imgcontent = io.Cache_from_filesystem(imgpath)
			} else {
				imgcontent = ret.GetData()
				isa.Remote_hit++
				// elapsed := time.Since(start)
				// log.Printf("time read a file from remote peer [%v]:", elapsed)
				clsidx := DCRuntime.Imgidx2clsidx[imgidx]
				return imgcontent, clsidx, false
			}
		} else {
			imgpath := DCRuntime.Imgidx2imgpath[imgidx]
			imgcontent = io.Cache_from_filesystem(imgpath)
		}
		// imgpath := DCRuntime.Imgidx2imgpath[imgidx]
		// imgcontent := io.Cache_from_filesystem(imgpath)
		// insert into cache
		isa.Insert(imgidx, imgcontent)
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

func (isa *Isa_cache_mng) AccessAtOnceISA(imgidx int64) ([]byte, int64, bool) {
	//
	isa.RLock()
	cache_entry, ok := isa.cache[imgidx]
	isa.RUnlock()
	if ok {
		return cache_entry.imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[isa-design1-cachemng.go] ip:", ip)
			// start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx_t, Imgidx: imgidx, FullAccess: false})
			if err != nil {
				log.Info("[isa-design1-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
				imgpath := DCRuntime.Imgidx2imgpath[imgidx]
				imgcontent = io.Cache_from_filesystem(imgpath)
			} else {
				imgcontent = ret.GetData()
				isa.Remote_hit++
				// elapsed := time.Since(start)
				// log.Printf("time read a file from remote peer: [%v]", elapsed)
				clsidx := DCRuntime.Imgidx2clsidx[imgidx]
				return imgcontent, clsidx, false
			}
		} else {
			imgpath := DCRuntime.Imgidx2imgpath[imgidx]
			imgcontent = io.Cache_from_filesystem(imgpath)
		}
		// imgpath := DCRuntime.Imgidx2imgpath[imgidx]
		// imgcontent := io.Cache_from_filesystem(imgpath)
		if len(isa.cache) < int(isa.Cache_size) {
			// log.Info("start insert")
			isa.Insert(imgidx, imgcontent)
		}
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

func (isa *Isa_cache_mng) GetRmoteHit() int64 {
	return isa.Remote_hit
}
