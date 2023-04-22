package io

import (
	"image"
	"image/draw"
	"main/common"
	"time"

	// "main/services"
	"os"
	"syscall"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	log "github.com/sirupsen/logrus"
)

var (
	BaseDir string
)

func Start() {
	log.Info("[Mod] IO module success")
	BaseDir = common.Config.Dcserver_data_path
}

/*
*

	function: read imgpath from filesystem and cache it to self.cache
	imgpath: img in tarfile
	imgidx: img id, used to return img cls target
	return: imgcontent, cls target
*/
func Cache_from_filesystem(imgpath string) []byte {

	// bypass kernel page cache; direct i/o
	// start := time.Now()
	imgcontent := ReadFileDirectIO(imgpath)

	time.Sleep(time.Duration(2) * time.Millisecond) // if no remote storage nodes, add this line to simulate the remote latency
	return imgcontent
}

func ReadImgData(fileName string) []byte {
	reader, _ := os.Open(fileName)
	defer reader.Close()
	img, _, _ := image.Decode(reader)
	rect := img.Bounds()
	rgba := image.NewRGBA(rect)
	draw.Draw(rgba, rect, img, rect.Min, draw.Src)
	//fmt.Printf("%v\n", rgba.Pix)
	return rgba.Pix
}

func ReadFileDirectIO(imgpath string) []byte {
	fileh, err := os.OpenFile(imgpath, os.O_RDONLY|syscall.O_DIRECT, 0666)
	if err != nil {
		log.Debugf("[IO] open file [%v] failed", imgpath)
	}
	defer fileh.Close()

	fileinfo, _ := fileh.Stat()
	filesize := fileinfo.Size()
	filesize = (filesize + 4095) / 4096 * 4096
	imgcontent := make([]byte, filesize)
	n, err := fileh.Read(imgcontent)
	if err != nil {
		log.Debug("[IO] Read file err ", err)
	}
	imgcontent = imgcontent[:n]
	return imgcontent
}
