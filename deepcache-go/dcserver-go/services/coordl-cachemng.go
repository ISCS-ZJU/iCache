package services

import (
	"context"
	"errors"
	"main/distkv"
	"main/io"
	"main/rpc/cache"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type coordlNode struct {
	KEY    int64
	RESULT []byte // string -> imgcontent pointer
}

// the real cache in user space
// key: imgidx; value: pointer to a link node contains imgcontent
type Coordl_cache_mng struct {
	sync.RWMutex
	Cache_size   int64                 `json:"cache_size"` // cache size
	cache        map[int64]*coordlNode // node pointer
	Full         bool                  `json:"full"`         // cache is full
	Cache_nitems int64                 `json:"cache_nitems"` // cache n items
	Remote_hit   int64                 `json:"remote_hit_times"`
}

func init_coordl_cache_mng(dc *deepcachemodel) *Coordl_cache_mng {
	var coor Coordl_cache_mng
	coor.Cache_size = dc.Cache_size
	coor.cache = make(map[int64]*coordlNode)
	coor.Full = false
	coor.Cache_nitems = 0

	return &coor
}

func (coor *Coordl_cache_mng) Exists(imgidx int64) bool {

	coor.Lock()
	defer coor.Unlock()

	_, ok := coor.cache[imgidx]
	if ok {
		return true
	} else {
		return false
	}
}

func (coor *Coordl_cache_mng) Access(imgidx int64) ([]byte, error) {

	coor.RLock()
	defer coor.RUnlock()
	// if in cache, return imgcontent and change metadata; if not in cache, raise keyerror exception;
	link, exist := coor.cache[imgidx]
	if exist {
		imgcontent := link.RESULT
		return imgcontent, nil
	} else {
		return nil, errors.New("not exist")
	}
}

func (coor *Coordl_cache_mng) Insert(imgidx int64, imgcontent []byte) error {
	// lock for cur write
	coor.Lock()
	defer coor.Unlock()
	// insert an element into self.cache
	// if not full, insert it
	// if full, do nothing
	if !coor.Full {
		link := &coordlNode{KEY: imgidx, RESULT: imgcontent}
		coor.cache[imgidx] = link

		if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
			log.Fatal("[lru-cachemng.go]", err.Error())
		} else {
			log.Infof("[lru-cachmng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
		}

		// update metadata
		coor.Cache_nitems += 1
		coor.Full = (coor.Cache_nitems >= coor.Cache_size)
	}
	return nil
}

func (coor *Coordl_cache_mng) Get_type() string {
	return "coordl"
}

func (coor *Coordl_cache_mng) AccessAtOnce(imgidx int64) ([]byte, int64, bool) {

	coor.RLock()
	link, ok := coor.cache[imgidx]
	coor.RUnlock()
	if ok {
		imgcontent := link.RESULT
		return imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[coordl-cachemng.go] ip:", ip)
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx, Imgidx: imgidx, FullAccess: false})
			if err != nil {
				log.Info("[coordl-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
				imgpath := DCRuntime.Imgidx2imgpath[imgidx]
				imgcontent = io.Cache_from_filesystem(imgpath)
			} else {
				imgcontent = ret.GetData()
				elapsed := time.Since(start)
				log.Debugf("time read a file from remote peer: [%v]", elapsed)
				coor.Remote_hit++
				clsidx := DCRuntime.Imgidx2clsidx[imgidx]
				return imgcontent, clsidx, false

			}
		} else {
			imgpath := DCRuntime.Imgidx2imgpath[imgidx]
			imgcontent = io.Cache_from_filesystem(imgpath)
		}
		// insert into cache
		coor.Insert(imgidx, imgcontent)
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

func (coor *Coordl_cache_mng) GetRmoteHit() int64 {
	return coor.Remote_hit
}
