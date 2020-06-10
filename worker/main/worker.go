package main


/*
worker
实现的功能 愿景
1. 从etcd中把job同步到内存当中
2. 实现调度模块,基于cron表达式调度N个job
3. 实现执行模块,并发到执行多个job任务
4. 对job的分布式锁，防止集群并发
5. 对执行日志保存到MongoDB当中
*/

import (
	"fmt"
	"runtime"
	"flag"
	"crontab/worker"
	"time"
)

var (
	confFile string // 配置文件路径
)

func initArg() {
	// wroker -config ./master.json
	// worker -h
	flag.StringVar(&confFile, "config", "./worker.json", "传入JSON配置文件")
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
	if err = worker.InitConfig(confFile); err != nil {
		goto ERR
	}
	// 初始化任务执行器
	if err = worker.InitExcuter();err != nil{
		goto ERR
	}
	// 初始化任务调度器
	if err = worker.InitScheduler(); err != nil{
		goto ERR
	}
	// 初始化任务管理器
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	// 阻塞
	for {
		time.Sleep(1 * time.Second)
	}


ERR:
	fmt.Println(err)

}
