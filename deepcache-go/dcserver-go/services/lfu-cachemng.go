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

type DoubleList struct {
	head, tail *Node
}

type Node struct {
	prev, next *Node
	key        int64  // imgidx
	val        []byte // img content
	freq       int64  // freq
}

func CreateDL() *DoubleList {
	head, tail := &Node{}, &Node{}
	head.next, tail.prev = tail, head
	return &DoubleList{
		head: head,
		tail: tail,
	}
}

func (this *DoubleList) AddFirst(node *Node) {
	node.next = this.head.next
	node.prev = this.head

	this.head.next.prev = node
	this.head.next = node
}

func (this *DoubleList) Remove(node *Node) {
	node.prev.next = node.next
	node.next.prev = node.prev

	node.next = nil
	node.prev = nil
}

func (this *DoubleList) RemoveLast() *Node {
	if this.IsEmpty() {
		return nil
	}

	last := this.tail.prev
	this.Remove(last)

	return last
}

func (this *DoubleList) IsEmpty() bool {
	return this.head.next == this.tail
}

// define Lfu_cache_mng
type Lfu_cache_mng struct {
	sync.RWMutex
	cache map[int64]*Node       // cache map
	freq  map[int64]*DoubleList // freq map

	Cache_size   int64 `json:"cache_size"`   // capacity
	Full         bool  `json:"full"`         // cache is full
	Cache_nitems int64 `json:"cache_nitems"` // cache n items
	Remote_hit   int64 `json:"remote_hit_times"`
	minFreq      int64 `json:"min Freq of all the items"`
}

func init_lfu_cache_mng(dc *deepcachemodel) *Lfu_cache_mng {
	var lfu Lfu_cache_mng
	lfu.Cache_size = dc.Cache_size
	lfu.cache = make(map[int64]*Node)
	lfu.Full = false
	lfu.Cache_nitems = 0

	lfu.freq = make(map[int64]*DoubleList)
	return &lfu
}

func (lfu *Lfu_cache_mng) Exists(imgidx int64) bool {

	lfu.Lock()
	defer lfu.Unlock()

	// if exists, return True (NOT change meta-data, because only access/insert will change), otherwise return False.
	_, ok := lfu.cache[imgidx]
	if ok {
		return true
	} else {
		return false
	}
}

func (lfu *Lfu_cache_mng) Access(imgidx int64) ([]byte, error) {
	// lock for cur write
	lfu.Lock()
	defer lfu.Unlock()

	// if in cache, return imgcontent and change metadata; if not in cache, raise keyerror exception;
	node, exist := lfu.cache[imgidx]
	if exist {
		imgcontent := node.val
		lfu.IncFreq(node)
		return imgcontent, nil
	} else {
		return nil, errors.New("not exist")
	}
}

func (lfu *Lfu_cache_mng) Insert(imgidx int64, imgcontent []byte) error {
	// lock for cur write
	lfu.Lock()
	defer lfu.Unlock()

	// insert an element into self.cache
	// if not full, insert it into the end
	// if full, insert it into root link node, and make the first link be new root()
	if lfu.Full {
		del_node := lfu.freq[lfu.minFreq].RemoveLast()
		oldkey := del_node.key
		delete(lfu.cache, oldkey)
		lfu.Cache_nitems--

		if err := distkv.GlobalKv.Del(oldkey); err != nil {
			log.Fatal("[lfu-cachemng.go]", err.Error())
		} else {
			log.Infof("[lfu-cachmng.go] deleted %v from server node %v", oldkey, distkv.Curaddr)
		}

	}

	new_node := &Node{key: imgidx, val: imgcontent, freq: 1}
	lfu.cache[imgidx] = new_node
	if lfu.freq[1] == nil {
		lfu.freq[1] = CreateDL()
	}
	lfu.freq[1].AddFirst(new_node)
	lfu.minFreq = 1
	lfu.Cache_nitems++

	lfu.Full = (lfu.Cache_nitems >= lfu.Cache_size)

	if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
		log.Fatal("[lfu-cachemng.go]", err.Error())
	} else {
		log.Infof("[lfu-cachmng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
	}

	return nil
}

func (lfu *Lfu_cache_mng) Get_type() string {
	return "lfu"
}

func (lfu *Lfu_cache_mng) AccessAtOnce(imgidx int64) ([]byte, int64, bool) {
	// lock for cur write
	// start := time.Now()
	lfu.RLock()
	node, exists := lfu.cache[imgidx]
	lfu.RUnlock()
	if exists {
		lfu.Lock()
		imgcontent := node.val
		lfu.IncFreq(node)
		// log.Infof("[%v] in cache, spent time [%v]", imgidx, time.Since(start))
		lfu.Unlock()
		return imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[lfu-cachemng.go] ip:", ip)
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx, Imgidx: imgidx, FullAccess: false, Id: 1})
			if err != nil {
				log.Info("[lfu-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
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
					lfu.Remote_hit++
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
		lfu.Insert(imgidx, imgcontent)

		// log.Infof("[%v] not in cache, INSERT spent time [%v]", imgidx, time.Since(start))
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

func (lfu *Lfu_cache_mng) GetRmoteHit() int64 {
	return lfu.Remote_hit
}

func (lfu *Lfu_cache_mng) IncFreq(node *Node) {
	_freq := node.freq
	lfu.freq[_freq].Remove(node)
	if lfu.minFreq == _freq && lfu.freq[_freq].IsEmpty() {
		lfu.minFreq++
		delete(lfu.freq, _freq)
	}
	node.freq++
	if lfu.freq[node.freq] == nil {
		lfu.freq[node.freq] = CreateDL()
	}
	lfu.freq[node.freq].AddFirst(node)
}
