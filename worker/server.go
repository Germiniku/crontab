package main

import (
	"crontab/worker"
	"fmt"
)

type IServer interface {
	InitConfig(string)
	InitExcuter()
	InitScheduler()
	InitJobMgr()
}


type Server struct {

}

var G_Server *Server

func NewServer()*Server{
	return G_Server
}

func (s *Server)Work(err error){
	// 初始化配置文件
	if err = InitConfig(confFile); err != nil {
		goto ERR
	}
	// 初始化任务执行器
	if err = InitExcuter();err != nil{
		goto ERR
	}
	// 初始化任务调度器
	if err = InitScheduler(); err != nil{
		goto ERR
	}
	// 初始化任务管理器
	if err = InitJobMgr(); err != nil {
		goto ERR
	}
ERR:
	fmt.Println(err)

}