package restful

import (
	"fmt"
	"main/common"
	"os"
	"os/exec"
	"strconv"
	"time"

	beegolog "github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/task"
	log "github.com/sirupsen/logrus"
)

func Start() {
	configbeego()
	// beegoadmin()
}

var FINISH_SERVER bool = false

type rootController struct {
	web.Controller
}

func (root *rootController) Get() {
	root.Ctx.WriteString("<html>" +
		"<title>DCServer</title>" +
		"<body>")
	root.Ctx.WriteString("<p>" + "welcome DeepCache Server (" + common.Config.Name + ")！</p>\n")
	root.Ctx.WriteString("<p><a target=\"_blank\" href=\"/cluster\">cluster info</a></p>")
	root.Ctx.WriteString("<p><a target=\"_blank\" href=\"/node\">node info</a></p>")
	root.Ctx.WriteString("<p><a target=\"_blank\" href=\"/cache\">cache info</a></p>")
	root.Ctx.WriteString("<p><a target=\"_blank\" href=\"/statistic\">statistic info</a></p>")
	root.Ctx.WriteString("<p><a target=\"_blank\" href=\"/mjob\">job info</a></p>")
	if common.Config.Enableadmin {
		root.Ctx.WriteString("<p><a target=\"_blank\" href=\"http://" +
			common.Config.Node + ":" + strconv.Itoa(common.Config.Restadminport) + "/qps\">request statistic</a></p>")
		root.Ctx.WriteString("<p><a target=\"_blank\" href=\"http://" +
			common.Config.Node + ":" + strconv.Itoa(common.Config.Restadminport) + "/metrics\">Prometheus</a></p>")
	}
	root.Ctx.WriteString("<p>Page generation time: " + time.Now().Format("2006-01-02 15:04:05") + "</p>\n")
	root.Ctx.WriteString("</body></html>")
}

func configbeego() {
	web.BConfig.AppName = common.Config.Name
	web.BConfig.Listen.HTTPAddr = common.Config.Node
	web.BConfig.Listen.HTTPPort = common.Config.Restport
	web.BConfig.Listen.AdminAddr = common.Config.Node
	web.BConfig.Listen.AdminPort = common.Config.Restadminport
	web.BConfig.Listen.EnableAdmin = common.Config.Enableadmin
	web.BConfig.CopyRequestBody = true
	beegolog.SetLevel(beegolog.LevelEmergency)

	web.Router("/cluster", &clusterController{})
	web.Router("/cache", &cacheController{})
	web.Router("/node", &nodeController{})
	web.Router("/statistic", &statisticController{})
	web.Router("/mjob", &multijobController{})
	web.Router("/*", &rootController{})
	task.StartTask()
	go web.Run()

	go exit_server()
}

func exit_server() {
	for {
		if FINISH_SERVER == true {
			out, _ := exec.Command("bash", "-c", "ps -ef | grep etcd | grep -v grep | awk '{print $2}' ").CombinedOutput()
			fmt.Println(string(out))
			_, err := exec.Command("bash", "-c", "kill "+string(out)).Output()
			fmt.Println(err)
			if err != nil {
				log.Fatal(err)
			} else {
				log.Info("Killed etcd server successfully.")
			}
			os.Exit(1)

		}
		time.Sleep(time.Duration(10) * time.Second)
	}
}
