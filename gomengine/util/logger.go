package util

import (
	"io"
	"log"
	"os"
)

var (
	Info *log.Logger
	Error *log.Logger
)

func init() {
	//日志输出文件
	file, err := os.OpenFile("order.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		log.Fatalln("Faild to open error logger file:", err)
	}
	//自定义日志格式
	Info = log.New(io.MultiWriter(file, os.Stderr), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(file, os.Stderr), "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}