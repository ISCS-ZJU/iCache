import requests
import time
import json

class MultijobMetaData():
    
    def __init__(self, REQ_HTTP_ADDR):
        self.http_endpoint = REQ_HTTP_ADDR+"mjob"

    # Send the request for the initialization of task information
    def multi_job_init(self):
        url = self.http_endpoint
        data = {"op":"INIT"}
        self.job_meta = requests.post(url, data).json()
        return

    # update task information to service node
    def multi_job_update_info(self, name, args):
        url = self.http_endpoint
        data = {
            "op":"UPDATE",
            "job_uuid":self.job_meta["job_uuid"],
            "job_name":name,
            "job_args":args
            }
        self.job_meta["job_name"] = name
        self.job_meta["job_args"] = args
        requests.put(url, data)
        return

    # updates stop timestamp to the service node
    def multi_job_stop(self):
        url = self.http_endpoint
        data = {
            "op":"FINISH",
            "job_uuid":self.job_meta["job_uuid"],
            }
        response = requests.put(url, data)
        self.job_meta["job_endtime"] = response.text
        return
    
    def __str__(self):
        return "[MJOB] " + json.dumps(self.job_meta)