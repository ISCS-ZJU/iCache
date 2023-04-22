package services

import (
	"sync"
)

// cache manager
type cache_mng_interface interface {
	Exists(int64) bool
	Access(int64) ([]byte, error)
	Insert(int64, []byte) error
	Get_type() string
	AccessAtOnce(int64) ([]byte, int64, bool)
	GetRmoteHit() int64
}

// isa-related parameters
type isa_related_struct struct {
	sync.RWMutex
	// Nupdatedsamples     int64             `json:"nupdatedsamples"`     // nupdatedsamples
	// Totalupdatedsamples int64             `json:"totalupdatedsamples"` // totalupdatedsamples
	// Current_epoch       int64             `json:"current_epoch"`       // current_epoch for regenerate important sample sequence
	// Important_samples   []int64           `json:"-"`                   // important_samples list
	// Is_in_ISL           []bool            `json:"-"`                   // 9,9: is important or not, both are in IL
	Ivpersample     map[int64]float64 `json:"-"` // importance value (iv) for each sample
	New_Ivpersample map[int64]float64 `json:"-"` // new importantce value (iv) for each sample
}

func init_isa_related_struct(dc *deepcachemodel) *isa_related_struct {
	isa_rs := isa_related_struct{}
	//importance value (iv) for each sample
	isa_rs.Ivpersample = make(map[int64]float64)
	isa_rs.New_Ivpersample = make(map[int64]float64)
	// important_samples can be duplicated
	// isa_rs.Important_samples = *(Shuffled_02ns_slice(dc.Noimg))
	// isa_rs.Is_in_ISL = make([]bool, dc.Noimg)
	for idx := int64(0); idx < dc.Noimg; idx++ {
		// isa_rs.Important_samples[idx] = idx // initialization: all samplesa are important samples
		isa_rs.Ivpersample[idx] = 1
		// isa_rs.Is_in_ISL[idx] = true
		isa_rs.New_Ivpersample[idx] = 1
	}

	//current epoch
	// isa_rs.Current_epoch = 0
	// samples' iv updated in the current epoch
	// isa_rs.Nupdatedsamples = 0
	// total samples' iv updated
	// isa_rs.Totalupdatedsamples = 0
	return &isa_rs
}

type heapNode struct {
	Imgidx int64 // imgidx
}

type indexheap struct {
	nodes             []heapNode
	Imgidx2importance *map[int64]float64
}

// functions necessary to build indexheap
// sort interface
func (h indexheap) Len() int { return int(len(h.nodes)) } // warning: must be int rather than int64
func (h indexheap) Less(i, j int) bool {
	return (*h.Imgidx2importance)[h.nodes[i].Imgidx] < (*h.Imgidx2importance)[h.nodes[j].Imgidx]
}
func (h indexheap) Swap(i, j int) {
	h.nodes[i], h.nodes[j] = h.nodes[j], h.nodes[i]
}

// heap interface push and pop
func (h *indexheap) Push(x interface{}) { (*h).nodes = append((*h).nodes, x.(heapNode)) }
func (h *indexheap) Pop() interface{} {
	old := (*h).nodes
	n := len(old)
	x := old[n-1]
	(*h).nodes = old[0 : n-1]
	return x
}
