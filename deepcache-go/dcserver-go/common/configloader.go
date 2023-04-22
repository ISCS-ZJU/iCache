package common

import (
	"fmt"
	"os"
	"time"

	"github.com/Pallinder/go-randomdata"
	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	"github.com/jinzhu/configor"
)

var Config = Configmodel{}
var Portpool = mapset.NewSet()

func configure(configpath string) {
	if configpath != "./conf/dcserver.yaml" {
		fmt.Println("config path:", configpath)
	}
	configor.Load(&Config, configpath)
	if Config.Name == "" {
		Config.Name = randomdata.SillyName()
	}
	if Config.Logpath != "" {
		fmt.Println("log path:", Config.Logpath)
	}
	if Config.Dcserver_data_path == "" {
		Config.Dcserver_data_path = "/data/dcserver/ufs"
	}

	u4 := uuid.New()
	if Config.Uuid == "" {
		Config.Uuid = u4.String()
	}

	Beego_bug_fix()
}

func Beego_bug_fix() {
	pwd, _ := os.Getwd()
	os.Mkdir(pwd+"/conf", 0776)
	path := pwd + "/conf/app.conf"
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0666)
	f.Close()
	time.Sleep(time.Second)
}
