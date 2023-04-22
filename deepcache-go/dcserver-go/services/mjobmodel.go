package services

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

type MultiJOB_Struct struct {
	Job_ID        int64  `json:"job_id"`        // job id
	Job_UUID      string `json:"job_uuid"`      // job uuid
	Job_Name      string `json:"job_name"`      // job name
	Job_Args      string `json:"job_args"`      // job args
	Job_StartTime string `json:"job_starttime"` // job start time, unix time
	Job_EndTime   string `json:"job_endtime"`   // job end time, unix time
}

var (
	Global_Job_Metadata      map[string]MultiJOB_Struct // metadata
	Global_Job_ID_GEN        int64
	Global_Job_metadata_lock sync.RWMutex
)

func multijob_init() {
	log.Debug("[MJOBSERVICES-INIT] init mjob module")
	Global_Job_Metadata = make(map[string]MultiJOB_Struct)
	Global_Job_ID_GEN = 1
}
