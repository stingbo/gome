package util

import (
	"io"
	"log"
	"os"
	"time"
)

var (
	Info *log.Logger
	Error *log.Logger
)
const (
	timeFormat = "2006-01-02 15:04:05"
)

func init() {
	//日志输出文件
	logFileName := "./logs/order-"+time.Now().Format(timeFormat)+".log"
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalln("Failed to open error logger file:", err)
	}
	//自定义日志格式
	Info = log.New(io.MultiWriter(file, os.Stderr), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(file, os.Stderr), "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}