package distkv

import (
	"context"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type raftkv struct {
	cli *clientv3.Client
	sync.RWMutex
}

func connect(ip string, port string) *raftkv {
	cfg := clientv3.Config{
		Endpoints: []string{ip + ":" + port},
		// Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		panic(err)
	}
	r := new(raftkv)
	r.cli = cli
	return r
}

func (r *raftkv) Put(imgid int64, nodeip string) error {
	r.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	kv := clientv3.NewKV(r.cli)
	_, err := kv.Put(ctx, strconv.FormatInt(imgid, 10), nodeip)
	cancel()
	r.Unlock()
	return err
}

func (r *raftkv) Get(imgid int64) (string, error) {
	r.RLock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	kv := clientv3.NewKV(r.cli)
	getResp, err := kv.Get(ctx, strconv.FormatInt(imgid, 10))
	r.RUnlock()
	cancel()
	if err != nil {
		log.Fatal("[kv.go]", err.Error())
		return "", err
	}
	if len(getResp.Kvs) == 0 {
		return "", err
	}
	return string(getResp.Kvs[0].Value), nil
}

func (r *raftkv) Del(imgid int64) error {
	r.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	kv := clientv3.NewKV(r.cli)
	_, err := kv.Delete(ctx, strconv.FormatInt(imgid, 10))
	cancel()
	r.Unlock()
	return err
}
