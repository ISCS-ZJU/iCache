package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"main/io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type quivernode struct {
	KEY    int64
	RESULT []byte // string -> imgcontent pointer
}

// the real cache in user space
// key: imgidx; value: pointer to a link node contains imgcontent

type Quiver_cache_mng struct {
	// sync.RWMutex
	Cache_size int64        `json:"cache_size"` // cache size
	cache      *Block_Cache `json:"-"`          // node pointer

	// channel
	blocks     chan int `json:"-"` // record 2 blocks
	block_size int64    `json:"-"` // block_size

	// double buffer
	next_cache *Block_Cache `json:"-"` // storing the next block

	// channel
	next_block_is_ready chan int `json:"-"` // next_block is ready

	Remote_hit int64 `json:"remote_hit_times"`
}

type Block_Cache struct {
	sync.RWMutex
	cache        map[int64]*quivernode `json:"-"`            // node pointer
	Cache_nitems int64                 `json:"cache_nitems"` // cache n items
	Full         bool                  `json:"full"`         // cache is full
}

var randseed int64 = 2022

func init_Quiver_cache_mng(dc *deepcachemodel) *Quiver_cache_mng {
	var quiver Quiver_cache_mng
	quiver.Cache_size = dc.Cache_size
	quiver.cache = new(Block_Cache)
	quiver.cache.cache = make(map[int64]*quivernode)
	quiver.next_cache = new(Block_Cache)
	quiver.next_cache.cache = make(map[int64]*quivernode)

	quiver.blocks = make(chan int, 2)
	quiver.next_block_is_ready = make(chan int)
	quiver.block_size = dc.Package_size

	return &quiver
}

func (quiver *Quiver_cache_mng) Exists(imgidx int64) bool {
	// cur access control
	quiver.cache.RLock()
	defer quiver.cache.RUnlock()
	log.Debug("get quiver.cache rlock")
	log.Debug("len of quiver cache:", len(quiver.cache.cache), quiver.cache.Cache_nitems)
	// if exists, return True otherwise return False.
	_, exists := quiver.cache.cache[imgidx]
	if exists {
		return true
	} else {
		return false
	}
}

func (quiver *Quiver_cache_mng) Access(imgidx int64) ([]byte, error) {
	// // cur access control
	quiver.cache.Lock()
	defer quiver.cache.Unlock()
	// check if imgidx in cache
	_, exists := quiver.cache.cache[imgidx]
	if exists {
		link := quiver.cache.cache[imgidx]
		// read imgcontent
		imgcontent := link.RESULT
		// delete it from cache
		delete(quiver.cache.cache, imgidx)
		// log.Info("[QUIVER CACHE MNG] A: len(quiver.cache) = ", len(quiver.cache.cache))
		quiver.cache.Cache_nitems -= 1
		if len(quiver.cache.cache) == 0 {
			<-quiver.next_block_is_ready
			// exchange quiver.cache and quiver.next_cache
			// quiver.cache, quiver.next_cache = quiver.next_cache, quiver.cache
			// trigger to put the next block
			quiver.next_cache.Lock()
			quiver.cache.cache = quiver.next_cache.cache
			quiver.next_cache.cache = make(map[int64]*quivernode)
			quiver.cache.Cache_nitems = quiver.next_cache.Cache_nitems
			quiver.next_cache.Cache_nitems = 0
			quiver.next_cache.Unlock()
			<-quiver.blocks
		}
		return imgcontent, nil

	} else {
		return nil, errors.New("not exist")
	}
}

func (quiver *Quiver_cache_mng) RandomAccess() ([]byte, int64, error) {
	// // cur access control，
	quiver.cache.Lock()
	defer quiver.cache.Unlock()

	// randomly choose a key in cache and return its content
	keys := reflect.ValueOf(quiver.cache.cache).MapKeys()

	if len(keys) == 0 {
		return nil, -1, nil
	}
	rand.Seed(time.Now().UnixNano())
	selected_key := keys[rand.Intn(len(keys))].Interface().(int64)
	// imgcontent, err := quiver.Access(selected_key)
	link := quiver.cache.cache[selected_key]
	// read imgcontent
	imgcontent := link.RESULT
	// delete it from cache
	delete(quiver.cache.cache, selected_key)
	// log.Info("[QUIVER CACHE MNG] RA: len(quiver.cache) = ", len(quiver.cache.cache))
	quiver.cache.Cache_nitems -= 1
	if len(quiver.cache.cache) == 0 {
		<-quiver.next_block_is_ready
		// exchange quiver.cache and quiver.next_cache
		// quiver.cache, quiver.next_cache = quiver.next_cache, quiver.cache
		quiver.next_cache.Lock()
		quiver.cache.cache = quiver.next_cache.cache
		quiver.next_cache.cache = make(map[int64]*quivernode)
		quiver.cache.Cache_nitems = quiver.next_cache.Cache_nitems
		quiver.next_cache.Cache_nitems = 0
		quiver.next_cache.Unlock()
		log.Debug("[QUIVER CACHE MNG] swap cache and next_cache complete")
		// trigger to put the next block
		<-quiver.blocks
	}
	// if err != nil {
	// 	log.Debugf("[QUIVER SERVER] read [%v] from quiver.cache failed", selected_key)
	// }
	return imgcontent, selected_key, nil
}

func (quiver *Quiver_cache_mng) AccessAtOnceQuiver(imgidx int64) ([]byte, int64) {
	quiver.cache.RLock()
	// quiver.cache.Lock()
	// defer quiver.cache.Unlock()
	// check if imgidx in cache
	link, exists := quiver.cache.cache[imgidx]
	quiver.cache.RUnlock()
	if exists {
		quiver.cache.Lock()
		defer quiver.cache.Unlock()
		// link := quiver.cache.cache[imgidx]
		// read imgcontent
		imgcontent := link.RESULT
		// delete it from cache
		delete(quiver.cache.cache, imgidx)
		// log.Info("[QUIVER CACHE MNG] A: len(quiver.cache) = ", len(quiver.cache.cache))
		quiver.cache.Cache_nitems -= 1
		if len(quiver.cache.cache) == 0 {
			<-quiver.next_block_is_ready
			// exchange quiver.cache and quiver.next_cache
			// quiver.cache, quiver.next_cache = quiver.next_cache, quiver.cache
			// trigger to put the next block
			quiver.next_cache.Lock()
			quiver.cache.cache = quiver.next_cache.cache
			quiver.next_cache.cache = make(map[int64]*quivernode)
			quiver.cache.Cache_nitems = quiver.next_cache.Cache_nitems
			quiver.next_cache.Cache_nitems = 0
			quiver.next_cache.Unlock()
			<-quiver.blocks
		}
		return imgcontent, imgidx

	} else {
		return nil, -1
	}
}

func (quiver *Quiver_cache_mng) Insert(imgidx int64, imgcontent []byte) error {
	// cur access control
	quiver.next_cache.Lock()
	defer quiver.next_cache.Unlock()
	NewQuiverNode := &quivernode{KEY: imgidx, RESULT: imgcontent}
	quiver.next_cache.cache[imgidx] = NewQuiverNode
	quiver.next_cache.Cache_nitems += 1
	log.Debugf("[QUIVER CACHE MNG] insert [%v] into quiver.next_cache done", imgidx)
	log.Debug("[QUIVER CACHE MNG] after insert len(quiver.next_cache) = ", len(quiver.next_cache.cache))
	return nil
}

func (quiver *Quiver_cache_mng) Get_type() string {
	return "quiver"
}

type quiver_entry struct {
	imgidx  int64
	imgpath string
}

func (quiver *Quiver_cache_mng) worker_loop(loadchan chan quiver_entry) {
	for {
		t_quiver_entry := <-loadchan
		imgidx := t_quiver_entry.imgidx
		imgcontent := Read_file_content(t_quiver_entry.imgpath)
		err := quiver.Insert(imgidx, imgcontent)
		if err != nil {
			log.Debug("[QUIVER CACHE MNG] insert imgidx:[%v] into dc.Cache_mng.cache error", imgidx)
		}
	}
}

func (quiver *Quiver_cache_mng) Quiver_cache_update_goroutine() {
	first_fill := false
	quiver.write_packs_to_disk()

	loadchan := make(chan quiver_entry, 1000)
	for i := 0; i < 6; i++ {
		go quiver.worker_loop(loadchan)
	}

	npackages := int64(math.Ceil(float64(DCRuntime.Noimg) / float64(DCRuntime.Package_size)))
	quiver_train_tars_folder_name := fmt.Sprintf("quiver_train_tars_logic_%v", quiver.Cache_size)
	// tardir := filepath.Join("/tmp/atc-quiver", quiver_train_tars_folder_name)
	tardir := filepath.Join(filepath.Dir(DCRuntime.Data_path), quiver_train_tars_folder_name)
	// quiver_train_tars_folder_name := ("quiver_train_tars")
	// tardir := filepath.Join("/mnt", quiver_train_tars_folder_name)

	// read tar and put samples into dc.Cache_mng.cache
	tar_slice := make([]int64, 0, npackages)
	for {
		quiver.blocks <- 1
		// quiver.Lock()
		log.Debugln("start to put new block...")
		// choose one tar file
		if len(tar_slice) == 0 {
			tar_slice = *(Shuffled_02ns_slice(npackages))
			log.Info("new tar slice:", tar_slice)
			// copy(last_tar_slice, tar_slice)
		}
		tar_id := tar_slice[0]
		log.Debugf("[SERVICE-QUIVER] tar_id %v ", tar_id)
		tar_name := fmt.Sprintf("%v.tar", tar_id)
		tar_path := filepath.Join(tardir, tar_name)
		if len(tar_slice) > 1 {
			tar_slice = tar_slice[1:]
		} else {
			tar_slice = make([]int64, 0, npackages)
		}

		dir_meta := make(map[int64]string)
		dir_meta_path := filepath.Join(tar_path, "dirmeta")
		jdata, _ := ioutil.ReadFile(dir_meta_path)
		json.Unmarshal(jdata, &dir_meta)

		for imgidx, idxpath := range dir_meta {
			loadchan <- quiver_entry{imgidx: imgidx, imgpath: idxpath}
		}

		// if quiver.cache is empty, exchange quiver.cache and quiver.next_cache
		if !first_fill {
			// exchange quiver.cache and quiver.next_cache
			// quiver.cache, quiver.next_cache = quiver.next_cache, quiver.cache
			quiver.next_cache.Lock()
			quiver.cache.cache = quiver.next_cache.cache
			quiver.next_cache.cache = make(map[int64]*quivernode)
			quiver.cache.Cache_nitems = quiver.next_cache.Cache_nitems
			quiver.next_cache.Cache_nitems = 0
			quiver.next_cache.Unlock()
			first_fill = true
		}
		log.Debug("put new block done.")
		quiver.next_cache.RLock()
		log.Debug("[QUIVER CACHE MNG] len(quiver.cache) = ", len(quiver.cache.cache))
		log.Debug("[QUIVER CACHE MNG] len(quiver.next_cache) = ", len(quiver.next_cache.cache))
		if int64(len(quiver.next_cache.cache)) > 0 {
			quiver.next_block_is_ready <- 1
		}
		quiver.next_cache.RUnlock()
	}
}

func Read_file_content(imgpath string) []byte {
	imgcontent := io.Cache_from_filesystem(imgpath)
	return []byte(imgcontent)
}

func Shuffled_02ns_slice(slice_len int64) *[]int64 {
	rt_slice := make([]int64, slice_len)
	for i := int64(0); i < int64(slice_len); i++ {
		rt_slice[i] = i
	}
	rand.Seed(randseed)
	log.Infoln("Shuffled_02ns_slice randseed:", randseed)
	randseed++
	rand.Shuffle(int(slice_len), func(i, j int) {
		rt_slice[i], rt_slice[j] = rt_slice[j], rt_slice[i]
	})
	return &rt_slice
}

func (quiver *Quiver_cache_mng) write_packs_to_disk() bool {

	// make packages according to cache_ratio
	npackages := int64(math.Ceil(float64(DCRuntime.Noimg) / float64(DCRuntime.Package_size)))
	total_imgs_in_tars := int64(npackages * DCRuntime.Package_size) // make sure each tar has the same size
	log.Info("total imgs in tars:", total_imgs_in_tars, ", total imgs:", DCRuntime.Noimg)
	// shuffle the whole dataset ID, pack them into #total_packages tars
	// write to disk
	global_slice := *(Shuffled_02ns_slice(DCRuntime.Noimg))

	// start to pack
	// make dir to store tars
	quiver_train_tars_folder_name := fmt.Sprintf("quiver_train_tars_logic_%v", quiver.Cache_size)
	tardir := filepath.Join(filepath.Dir(DCRuntime.Data_path), quiver_train_tars_folder_name)
	// tardir := filepath.Join("/tmp/atc-quiver", quiver_train_tars_folder_name)
	// quiver_train_tars_folder_name := ("quiver_train_tars")
	// tardir := filepath.Join("/mnt", quiver_train_tars_folder_name)
	_, err := os.Stat(tardir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(tardir, 0755)
		if errDir != nil {
			log.Debugf("[SERVICE-QUIVER] Create [%v] failed.", tardir)
		}
	}

	for pack_id := int64(0); pack_id < npackages; pack_id++ {
		// create one tarball file
		target := filepath.Join(tardir, fmt.Sprintf("%v.tar", pack_id))
		_, err := os.Stat(target)
		if os.IsExist(err) {
			os.RemoveAll(target)
		}
		errDir := os.MkdirAll(target, 0755)
		if errDir != nil {
			log.Debugf("[SERVICE-QUIVER] Create [%v] failed.", target)
		}
		dir_meta := make(map[int64]string)
		dir_meta_path := filepath.Join(target, "dirmeta")
		for _, idx := range global_slice[pack_id*DCRuntime.Package_size : int64(math.Min(float64((pack_id+1)*DCRuntime.Package_size), float64(len(global_slice))))] {
			dir_meta[idx] = DCRuntime.Imgidx2imgpath[idx]
		}
		jdata, _ := json.Marshal(dir_meta)
		err = ioutil.WriteFile(dir_meta_path, jdata, 0644)
		if err != nil {
			log.Warnf("[SERVICE-QUIVER] Create [%v] failed, %v", dir_meta_path, err)
		}
	}

	log.Infof("[SERVICE-QUIVER] tar files done. Total [%v] tarballs.", npackages)
	return true
}

func (quiver *Quiver_cache_mng) AccessAtOnce(imgidx int64) ([]byte, int64, bool) {
	return nil, -1, false
}

func (quiver *Quiver_cache_mng) GetRmoteHit() int64 {
	return quiver.Remote_hit
}
