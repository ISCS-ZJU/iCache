# iCache: An Importance-Sampling-Informed Cache for Accelerating I/O-Bound DNN Model Training

This is the codebase for the [paper](https://ieeexplore.ieee.org/abstract/document/10070964/) that was accepted at *HPCA 2023*.

## Table of Contents
- [Code Architecture](#code-architecture)
- [Prerequisites and Installation](#prerequisites-and-installation)
- [Getting Started](#getting-started)
    - [Configurations](#configurations)
    - [Single node DNN training](#single-node-dnn-training)
    - [Distributed DNN training](#distributed-dnn-training)

## Code Architecture
All the code is stored in the `deepcache-go` directory, with two subdirectories: `dcclient-python` and `dcserver-go`. `dcclient-python` is a modified version of the PyTorch framework used for deep neural network training, while `dcserver-go` is a Golang-based, distributed training data caching system that responds to client requests. To improve code structure and readability, we've separated different functions into their respective subdirectories.

## Prerequisites and Installation
To get started with this project, please follow these steps to to obtain the code and install the necessary packages.
1. Clone the repository to your local machine and navigate to the project directory.
    ```bash
    git clone https://github.com/ISCS-ZJU/iCache
    cd iCache
    ```
2. Create conda env and install python dependencies.
    ```bash
    conda create -n icache python==3.9 # create conda env
    conda activate icache # activate env
    pip3 install torch==1.8.0+cu111 torchvision==0.9.0+cu111 -f https://download.pytorch.org/whl/torch_stable.html
    pip3 install six grpcio==1.46.1 grpcio-tools==1.46.1 requests==2.27.1 # installing dependencies
    ```
3. Install golang and turn on go module function.
    ```bash
    wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin # add this line at the end of your ~/.bashrc file
    source ~/.bashrc # apply the changes immediately
    go version # verify if golang has been installed successfully.
    go env -w GO111MODULE=on # turn on go module 
    ```
4. To prepare for training, you will need to install a remote parallel file system (PFS) on the storage nodes and store the training data on the PFS. Make sure the computing nodes can access the training data on the storage nodes.

    **NOTE:** *In our paper, we used OrangeFS as the parallel file system. However, depending on your specific environment, you may choose to use other alternative parallel file systems. To install OrangeFS, please refer to the detailed installation process outlined in [this link](https://docs.orangefs.com/quickstart/quickstart-build/).*

5. Because we have extended iCache into distributed version, you also need to install `etcd` served as the distributed KV store. To do this, please download `etcd-3.4.4-linux-amd64` from [here](https://github.com/etcd-io/etcd/releases/download/v3.4.4/etcd-v3.4.4-linux-amd64.tar.gz), extract the contents of the archive, and add the path to the `etcd` executable files to your PATH environment variable.

## Getting Started
### Configurations
1. For the python client side, modify related files in PyTorch frameworks.
    ```bash
    cd deepcache-go/dcclient-python/
    python3 prep-1.8.py
    ```
2. Enable python and golang communication by re-generating grcp files.
    ```bash
    cd deepcache-go/dcclient-python/dcrpc
    
    protoc \
    --go_out=Mgrpc/service_config/service_config.proto=/internal/proto/grpc_service_config:.\
    --go-grpc_out=Mgrpc/service_config/service_config.proto=/internal/proto/grpc_service_config:.\
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    deepcache.proto
    
    python -m grpc_tools.protoc --python_out=. --grpc_python_out=. -I. deepcache.proto
    
    cd deepcache-go/dcserver-go/rpc/cache/ # and execute the same above two commands
    ```
3. Configure the cache server yaml file `conf/dcserver.yaml` in `deepcache-go/dcserver-go/conf`.

    **NOTE:** *When running DNN training on a single computing node, enter the IP address of the current node in the etcd_nodes field. For distributed DNN training across multiple nodes, enter the IP addresses separated by commas in the etcd_nodes field.*

### Single node DNN training
To run iCache on a single computing node, please follow these steps:
1. Start etcd on single node.
    ```bash
    rm -rf etcd_name0.etcd && etcd --name etcd_name0 --listen-peer-urls http://{node_ip}:2380 --initial-advertise-peer-urls http://{node_ip}:2380 --listen-client-urls http://{node_ip}:2379 --advertise-client-urls http://{node_ip}:2379 --initial-cluster etcd_name0=http://{node_ip}:2380,
    ```
2. Start the cache server.
    ```bash
    cd deepcache-go/dcserver-go
    go run dcserver.go # you can use -config yaml_file_path to specify the target yaml file to run
    ```
3. Open another terminal and start the client DNN training.
    ```bash
    cd deepcache-go/dcclient-python
    CUDA_VISIBLE_DEVICES=0 time python3 -u cached_image_classification.py --arch resnet18 --epochs 90 -b 256 --worker 2 -p 20 --num-classes 10 --req-addr http://127.0.0.1:18283/ --grpc-port {node_ip}:18284 # if the cache_type in yaml is isa, you should add `--sbp --reuse-factor 3 --warm-up 5` flags to specify the tunable hyper-parameters used by the importance sampling algorithm.
    ```

### Distributed DNN training
To run iCache on multiple computing nodes (we will use two nodes with IP addresses `ip0` and `ip1` as an example), please follow these steps:
1. Start a distributed KV on each node.
    ```bash
    rm -rf etcd_name0.etcd && etcd --name etcd_name0 --listen-peer-urls http://{ip0}:2380 --initial-advertise-peer-urls http://{ip0}:2380 --listen-client-urls http://{ip0}:2379 --advertise-client-urls http://{ip0}:2379 --initial-cluster etcd_name1=http://{ip1}:2380,etcd_name0=http://{ip0}:2380,

    rm -rf etcd_name1.etcd && etcd --name etcd_name1 --listen-peer-urls http://{ip1}:2380 --initial-advertise-peer-urls http://{ip1}:2380 --listen-client-urls http://{ip1}:2379 --advertise-client-urls http://{ip1}:2379 --initial-cluster etcd_name1=http://{ip1}:2380,etcd_name0=http://{ip0}:2380,
    ````
2. Start two cache servers on each nodes. Make sure the two yaml configuration files are the same.
3. Open two client terminals on each node and start clients.
    ```bash
    # rank 0
    CUDA_VISIBLE_DEVICES=0 python3 -u cached_image_classification.py --arch resnet18 --epochs 5 --worker 2 -b 256 -p 20 --num-classes 10 --seed 2022 --multiprocessing-distributed --world-size 2 --rank 0 --ngpus 1 --dist-url 'tcp://{ip0}:1234'  --req-addr http://127.0.0.1:18283/ --grpc-port {ip0}}:18284
    # rank 1
    CUDA_VISIBLE_DEVICES=1 python3 -u cached_image_classification.py --arch resnet18 --epochs 5 --worker 2 -b 256 -p 20 --num-classes 10 --seed 2022 --multiprocessing-distributed --world-size 2 --rank 1 --ngpus 1 --dist-url 'tcp://{ip0}:1234'  --req-addr http://127.0.0.1:18283/ --grpc-port {ip1}:18284
    # if the cache_type is isa, please add flags as stated previously.
    ```
