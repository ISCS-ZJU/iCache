import requests
import time
import json

class ClusterData():
    
    def __init__(self, REQ_HTTP_ADDR):
        self.http_endpoint = REQ_HTTP_ADDR+"cluster" # http://127.0.0.1:18283/cluster

    # Send the request for the initialization of task information
    def nodes_info_get(self):
        response = requests.get(self.http_endpoint)
        self.cluster_info = response.json()
    
    def __str__(self):
        return "[M_NODE] " + json.dumps(self.cluster_info)

if __name__ == '__main__':
    cc = ClusterData("http://127.0.0.1:18283/")
    cc.nodes_info_get()
    print(cc)