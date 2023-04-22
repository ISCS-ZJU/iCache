package distkv

import (
	"main/common"
	"main/rpc/cache"
	"net"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	GlobalKv    *raftkv
	Curaddr     string
	GrpcClients map[string]cache.OperatorClient
)

func Start() {
	// start etcd
	/*
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Fatal("[distkv.go] Get IP addr err" + err.Error())
		}
		var Curaddr string
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					Curaddr = ipnet.IP.String()
				}
			}
		}
		ip_slice := strings.Split(Curaddr, ".")
		ip_last := ip_slice[len(ip_slice)-1]
		log.Debug("ip_last:", ip_last, "cur_ip:", Curaddr)
		etcd_nodes := common.Config.Etcd_nodes
		ips_slice := strings.Split(etcd_nodes, ",")
		name := "etcd" + ip_last
		server_ip_port := "http://" + Curaddr + ":2380"
		client_ip_port := "http://" + Curaddr + ":2379"
		var etcd_cmd = `etcd --name etcd%s \
						--listen-peer-urls http://%s:2380 \
						--initial-advertise-peer-urls http://%s:2380 \
						--listen-client-urls http://%s:2379 \
						--advertise-client-urls http://%s:2379 \
						--initial-cluster `
		etcd_cmd = fmt.Sprintf(etcd_cmd, ip_last, Curaddr, Curaddr, Curaddr, Curaddr)
		fmt.Println(etcd_cmd)
		cluster_ips_ports := ""
		for _, addr := range ips_slice {
			tmp_ip_slice := strings.Split(addr, ".")
			tmp_ip_last := tmp_ip_slice[len(tmp_ip_slice)-1]
			cluster_ips_ports += "etcd" + tmp_ip_last + "=http://" + addr + ":2380,"
		}
		log.Info("[distkv.go]", cluster_ips_ports)

		go run_etcd_cmd(name, server_ip_port, client_ip_port, cluster_ips_ports)
		time.Sleep(time.Duration(5) * time.Second)
	*/

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal("[distkv.go] Get IP addr err" + err.Error())
	}
	log.Info("[distkv.go] addrs:", addrs)

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				Curaddr = ipnet.IP.String()
				break
			}
		}
	}
	log.Info("[distkv.go] Curaddr:", Curaddr)
	GlobalKv = connect(Curaddr, "2379")
	log.Infoln("[distkv.go] Connected to Etcd server.")
	etcd_nodes := common.Config.Etcd_nodes
	ips_slice := strings.Split(etcd_nodes, ",")
	log.Infoln("[distkv.go] ips_slice:", ips_slice)

	GrpcClients = make(map[string]cache.OperatorClient)
	for _, addr := range ips_slice {
		log.Infof("[distkv.go] %v %T %v %T", addr, addr, Curaddr, Curaddr)
		if addr != Curaddr {
			for {
				log.Infoln("[distkv.go] start to connect to grpc server", addr+":"+strconv.Itoa(common.Config.Rpcport))
				conn, err := grpc.Dial(addr+":"+strconv.Itoa(common.Config.Rpcport), grpc.WithInsecure(), grpc.WithBlock())
				if err != nil {
					log.Fatalf("[distkv.go] Failed to connect: %v", err)
				} else {
					log.Infof("[distkv.go] Connected to Grpc server %v", addr)
					GrpcClients[addr] = cache.NewOperatorClient(conn)
					break
				}
			}
		}
	}

	log.Infoln("[distkv.go] Connected to Grpc servers.")
	for k, v := range GrpcClients {
		log.Debugln(k, "'s value is", v)
	}

}

/*
func run_etcd_cmd(name, server_ip_port, client_ip_port, cluster_ips_ports string) {

	cmd := exec.Command("etcd", "--name", name, "--listen-peer-urls", server_ip_port, "--initial-advertise-peer-urls", server_ip_port, "--listen-client-urls", client_ip_port, "--advertise-client-urls", client_ip_port, "--initial-cluster", cluster_ips_ports)
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		log.Fatal("[distkv.go]", err.Error())
	} else {
		log.Info("[distkv.go] ETCD server success √")
	}
}
*/

// func main() {
// 	Start()
// }
