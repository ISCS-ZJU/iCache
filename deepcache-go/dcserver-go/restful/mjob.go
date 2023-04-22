package restful

import (
	"encoding/json"
	"fmt"
	multijobservices "main/services"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"
	// log "github.com/sirupsen/logrus"
)

type multijobController struct {
	web.Controller
}

func (mjob *multijobController) Get() {
	mjobsmeta, _ := json.Marshal(multijobservices.Global_Job_Metadata)
	mjob.Ctx.WriteString(string(pretty.Pretty(mjobsmeta)))
}

func (mjob *multijobController) Post() {
	op_type := mjob.GetString("op")
	if op_type == "INIT" {
		multijobservices.Global_Job_metadata_lock.Lock()
		jobmeta := multijobservices.MultiJOB_Struct{}
		jobmeta.Job_ID = multijobservices.Global_Job_ID_GEN
		jobmeta.Job_UUID = uuid.New().String()
		jobmeta.Job_StartTime = fmt.Sprint(time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")) + " " + fmt.Sprint(time.Now().UnixNano())
		// jobmeta.Job_StartTime = fmt.Sprint(time.Now().UnixNano())
		multijobservices.Global_Job_ID_GEN++
		multijobservices.Global_Job_Metadata[jobmeta.Job_UUID] = jobmeta
		multijobservices.Global_Job_metadata_lock.Unlock()
		mjobmeta, _ := json.Marshal(jobmeta)
		mjob.Ctx.WriteString(string(pretty.Pretty(mjobmeta)))
		return
	}

	if op_type == "GETBYUUID" {
		request_job_uuid := mjob.GetString("uuid")
		log.Debug("[REST-MJOB-Post] UUID: ", request_job_uuid)
		multijobservices.Global_Job_metadata_lock.RLock()
		meta, exist := multijobservices.Global_Job_Metadata[request_job_uuid]
		if exist {
			mjobmeta, _ := json.Marshal(meta)
			mjob.Ctx.WriteString(string(pretty.Pretty(mjobmeta)))
		} else {
			mjob.Ctx.WriteString("UUID of target job not exists" + request_job_uuid)
		}
		multijobservices.Global_Job_metadata_lock.RUnlock()
		return
	}

	mjob.Ctx.WriteString(string(op_type) + " not supported.")
}

func (mjob *multijobController) Put() {
	op_type := mjob.GetString("op")
	if op_type == "UPDATE" {
		uuid := mjob.GetString("job_uuid")
		job_name := mjob.GetString("job_name")
		job_args := mjob.GetString("job_args")
		multijobservices.Global_Job_metadata_lock.Lock()
		mm := multijobservices.Global_Job_Metadata[uuid]
		mm.Job_Name = job_name
		mm.Job_Args = job_args
		multijobservices.Global_Job_Metadata[uuid] = mm
		multijobservices.Global_Job_metadata_lock.Unlock()
		return
	}

	if op_type == "FINISH" {
		uuid := mjob.GetString("job_uuid")
		// endtime := fmt.Sprint(time.Now().UnixNano())
		endtime := fmt.Sprint(time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")) + " " + fmt.Sprint(time.Now().UnixNano())
		multijobservices.Global_Job_metadata_lock.Lock()
		mm := multijobservices.Global_Job_Metadata[uuid]
		mm.Job_EndTime = endtime
		multijobservices.Global_Job_Metadata[uuid] = mm
		multijobservices.Global_Job_metadata_lock.Unlock()
		mjob.Ctx.WriteString(endtime)

		FINISH_SERVER = true

		return
	}

	mjob.Ctx.WriteString(string("MJOB-Put"))
}

func (mjob *multijobController) Delete() {
	multijobservices.Global_Job_metadata_lock.Lock()
	len := len(multijobservices.Global_Job_Metadata)
	multijobservices.Global_Job_Metadata = make(map[string]multijobservices.MultiJOB_Struct)
	multijobservices.Global_Job_metadata_lock.Unlock()
	mjob.Ctx.WriteString(fmt.Sprintf("clear %v job meta data", len))
}
