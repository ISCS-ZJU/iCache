package services

import (
	log "github.com/sirupsen/logrus"
)

// design 2
type seqnode struct {
	KEY    int64
	RESULT []byte
}

type packaging_struct struct {
	// sync.Mutex
	// sequential buffer for design 2
	UC            chan seqnode `json:"-"`       // UC cache
	Loading_Cache chan seqnode `json:"-"`       // Loading_Cache
	Nimages       int64        `json:"nimages"` // image num
}

func init_packaging_struct(isa *Isa_cache_mng) *packaging_struct {
	ps := packaging_struct{}
	ps.UC = make(chan seqnode, isa.UC_size)
	ps.Loading_Cache = make(chan seqnode, isa.Loading_cache_size)
	ps.Nimages = DCRuntime.Noimg

	return &ps
}

func (ps *packaging_struct) make_global_shuffled_slice() []int64 {
	return *(Shuffled_02ns_slice(DCRuntime.Noimg))
}

func (ps *packaging_struct) worker_loop(slice_chan chan int64) {
	for {
		imgidx := <-slice_chan
		imgcontent := Read_file_content(DCRuntime.Imgidx2imgpath[imgidx])
		ps.Loading_Cache <- seqnode{KEY: imgidx, RESULT: imgcontent}
	}
}

func (ps *packaging_struct) loading_thread() {
	// TODO: start 4 worker
	slice_chan := make(chan int64, 10000)
	for i := 0; i < 4; i++ {
		go ps.worker_loop(slice_chan)
	}
	global_slice := ps.make_global_shuffled_slice()
	for {
		log.Debugf("[CACHE-ISA-LOADING] GlobalSlice %v Len %v", global_slice, len(global_slice))

		for _, package_id := range global_slice {
			slice_chan <- int64(package_id)
		}
		global_slice = ps.make_global_shuffled_slice()
		log.Debugf("[CACHE-ISA-LOADING] ReShuffle GlobalSlice %v Len %v", global_slice, len(global_slice))
	}
}

func (ps *packaging_struct) uc_thead() {
	for {
		t_seq_node := <-ps.Loading_Cache
		ps.UC <- t_seq_node
	}
}

func (ps *packaging_struct) RandomAccess() ([]byte, int64, error) {
	var t_seq_node seqnode
	var selected_key int64
	for {
		t_seq_node = <-ps.UC
		selected_key = t_seq_node.KEY
		break
	}
	imgcontent := t_seq_node.RESULT

	return imgcontent, selected_key, nil
}
