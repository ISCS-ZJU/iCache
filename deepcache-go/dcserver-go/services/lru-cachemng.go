package services

import (
	"context"
	"errors"
	"main/distkv"
	"main/io"
	"main/rpc/cache"
	"sync"
	"time"

	// "time"
	log "github.com/sirupsen/logrus"
)

type node struct {
	PREV   *node
	NEXT   *node
	KEY    int64
	RESULT []byte // string -> imgcontent pointer
}

// the real cache in user space
// key: imgidx; value: pointer to a link node contains imgcontent

type Lru_cache_mng struct {
	sync.RWMutex
	Cache_size   int64           `json:"cache_size"` // cache size
	cache        map[int64]*node // node pointer
	root         *node           // necessary meta-data to execute LRU cache replacement algorithm
	Full         bool            `json:"full"`         // cache is full
	Cache_nitems int64           `json:"cache_nitems"` // cache n items
	Remote_hit   int64           `json:"remote_hit_times"`
}

func init_lru_cache_mng(dc *deepcachemodel) *Lru_cache_mng {
	var lru Lru_cache_mng
	lru.Cache_size = dc.Cache_size
	lru.cache = make(map[int64]*node)
	lru.Full = false
	lru.Cache_nitems = 0

	// init root
	lru.root = &node{}
	lru.root.PREV = lru.root
	lru.root.NEXT = lru.root
	lru.root.KEY = -1
	lru.root.RESULT = nil
	return &lru
}

func (lru *Lru_cache_mng) Exists(imgidx int64) bool {

	lru.Lock()
	defer lru.Unlock()

	// if exists, return True (NOT change meta-data, because only access/insert will change), otherwise return False.
	_, ok := lru.cache[imgidx]
	if ok {
		return true
	} else {
		return false
	}
}

func (lru *Lru_cache_mng) Access(imgidx int64) ([]byte, error) {
	// lock for cur write
	lru.Lock()
	defer lru.Unlock()

	// if in cache, return imgcontent and change metadata; if not in cache, raise keyerror exception;
	link, exist := lru.cache[imgidx]
	if exist {
		// take this link away and read its imgcontent
		link_prev := link.PREV
		link_next := link.NEXT
		imgcontent := link.RESULT
		link_prev.NEXT = link_next
		link_next.PREV = link_prev

		// insert this link into last
		last := lru.root.PREV
		last.NEXT = link
		lru.root.PREV = link
		link.PREV = last
		link.NEXT = lru.root
		return imgcontent, nil
	} else {
		return nil, errors.New("not exist")
	}
}

func (lru *Lru_cache_mng) Insert(imgidx int64, imgcontent []byte) error {
	// lock for cur write
	lru.Lock()
	defer lru.Unlock()

	// insert an element into self.cache
	// if not full, insert it into the end
	// if full, insert it into root link node, and make the first link be new root()
	if !lru.Full {
		last := lru.root.PREV
		link := &node{PREV: last, NEXT: lru.root, KEY: imgidx, RESULT: imgcontent}
		last.NEXT = link
		lru.root.PREV = link
		lru.cache[imgidx] = link

		// update metadata
		lru.Cache_nitems += 1
		lru.Full = (lru.Cache_nitems >= lru.Cache_size)
	} else {
		// Use the old root to store the new key and result.
		oldroot := lru.root
		oldroot.KEY = imgidx
		oldroot.RESULT = imgcontent
		// Empty the oldest link and make it the new root.
		lru.root = oldroot.NEXT
		oldkey := lru.root.KEY
		lru.root.KEY = -1
		lru.root.RESULT = nil
		// Now update the cache dictionary.
		delete(lru.cache, oldkey)
		lru.cache[imgidx] = oldroot
		// total number keeps the same, so don't need modify metadata any more
		if err := distkv.GlobalKv.Del(oldkey); err != nil {
			log.Fatal("[lru-cachemng.go]", err.Error())
		} else {
			log.Infof("[lru-cachmng.go] deleted %v from server node %v", oldkey, distkv.Curaddr)
		}

	}
	if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
		log.Fatal("[lru-cachemng.go]", err.Error())
	} else {
		log.Infof("[lru-cachmng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
	}

	return nil
}

func (lru *Lru_cache_mng) Get_type() string {
	return "lru"
}

func (lru *Lru_cache_mng) AccessAtOnce(imgidx int64) ([]byte, int64, bool) {
	// lock for cur write
	// start := time.Now()
	lru.RLock()
	link, exists := lru.cache[imgidx]
	lru.RUnlock()
	if exists {
		lru.Lock()
		// take this link away and read its imgcontent
		link_prev := link.PREV
		link_next := link.NEXT
		start := time.Now()
		imgcontent := link.RESULT
		elapsed := time.Since(start)
		log.Printf("time read a file from local-cache: [%v]", elapsed)
		link_prev.NEXT = link_next
		link_next.PREV = link_prev

		// insert this link into last
		last := lru.root.PREV
		last.NEXT = link
		lru.root.PREV = link
		link.PREV = last
		link.NEXT = lru.root
		// log.Infof("[%v] in cache, spent time [%v]", imgidx, time.Since(start))
		lru.Unlock()
		return imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[lru-cachemng.go] ip:", ip)
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx, Imgidx: imgidx, FullAccess: false, Id: 1})
			if err != nil {
				log.Info("[lru-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
				start := time.Now()
				imgpath := DCRuntime.Imgidx2imgpath[imgidx]
				imgcontent = io.Cache_from_filesystem(imgpath)
				elapsed := time.Since(start)
				log.Infof("time read a file from file system: [%v]", elapsed)
				if len(imgcontent) == 0 {
					log.Error("0 read a NULL file from filesystem for imgidx [%v]", imgidx)
				}
			} else {
				imgcontent = ret.GetData()
				if len(imgcontent) == 0 {
					log.Info("read a null from remote peer cache")
					start := time.Now()
					imgpath := DCRuntime.Imgidx2imgpath[imgidx]
					imgcontent = io.Cache_from_filesystem(imgpath)
					elapsed := time.Since(start)
					log.Infof("time read a file from file system: [%v]", elapsed)
					if len(imgcontent) == 0 {
						log.Error("1 read a NULL file from filesystem for imgidx [%v]", imgidx)
					}
					clsidx := DCRuntime.Imgidx2clsidx[imgidx]
					return imgcontent, clsidx, false
				} else {
					elapsed := time.Since(start)
					log.Infof("time read a file from remote peer: [%v]", elapsed)
					lru.Remote_hit++
					clsidx := DCRuntime.Imgidx2clsidx[imgidx]
					return imgcontent, clsidx, false
				}
			}
		} else {
			start := time.Now()
			imgpath := DCRuntime.Imgidx2imgpath[imgidx]
			imgcontent = io.Cache_from_filesystem(imgpath)
			elapsed := time.Since(start)
			log.Infof("time read a file from file system: [%v]", elapsed)
			if len(imgcontent) == 0 {
				log.Error("2 read a NULL file from filesystem for imgidx [%v]", imgidx)
			}
			// log.Infof("[%v] not in cache, IO file spent time [%v]", imgidx, time.Since(start))
		}
		// imgpath := DCRuntime.Imgidx2imgpath[imgidx]
		// imgcontent = io.Cache_from_filesystem(imgpath)

		// insert into cache
		// start = time.Now()
		lru.Insert(imgidx, imgcontent)

		// log.Infof("[%v] not in cache, INSERT spent time [%v]", imgidx, time.Since(start))
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

func (lru *Lru_cache_mng) GetRmoteHit() int64 {
	return lru.Remote_hit
}
