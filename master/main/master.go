package main



import (
	"crontab/master"
	"flag"
	"fmt"
	"runtime"
	"time"
)

/*
实现的功能:
	1.实现可配置化，命令实现配置文件加载
	2.给web后台提供http API,用于管理定时任务
	TODO: 实现web后台的前端页面
 */

var (
	confFile string // 配置文件路径
)

func initArg() {
	// master-config ./master.json
	flag.StringVar(&confFile, "config", "./master.json", "传入JSON配置文件")
	flag.Parse()
}

// 初始化线程
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)
	// 初始化线程
	initEnv()
	// 初始化命令行参数
	initArg()
	// 初始化配置文件
	if err = master.InitConfig(confFile); err != nil {
		goto ERR
	}
	// 初始化任务管理器
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}
	// 启动Api:HTTP服务
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	// 阻塞
	for {
		time.Sleep(1 * time.Second)
	}

	return

ERR:
	fmt.Println(err)
}
