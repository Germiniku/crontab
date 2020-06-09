package master

import (
	"crontab/common"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	// 单例
	G_apiserver *ApiServer
)

// 初始化服务
func InitApiServer() (err error) {
	var (
		mux        *http.ServeMux
		listener   net.Listener
		httpServer *http.Server
		port       string
	)
	// 配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/jobs/save", handleJobSave)
	mux.HandleFunc("/jobs/delete", handleJobDelete)
	mux.HandleFunc("/jobs/list", handleJobList)
	mux.HandleFunc("/jobs/kill", handleJobKill)
	// 建立TCP监听
	port = ":" + strconv.Itoa(G_config.ApiPort)
	if listener, err = net.Listen("tcp", port); err != nil {
		return
	}
	// 创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	G_apiserver = &ApiServer{
		httpServer: httpServer,
	}
	// 启动服务端
	go httpServer.Serve(listener)
	return
}

// 保存任务服务
// POST job={"name":"job1","command":"echo helloworld","cronExpr":"* * * * * *"}
func handleJobSave(w http.ResponseWriter, r *http.Request) {
	// 任务保存到etcd
	var (
		postJob string
		job     common.CronJob
		err     error
		oldjob  *common.CronJob
		resp    []byte
	)
	// 1.解析表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	// 2.获取表单信息
	postJob = r.PostForm.Get("job")
	log.Println(postJob)
	// 3.反序列化CronJob
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}
	// 4.保存到etcd中
	if oldjob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	// 5.返回正常应答

	//请求正常
	if resp, err = common.NewResponse(1, "success", oldjob); err == nil {
		w.Write(resp)
	}

	return

ERR:
	// 请求异常
	if resp, err = common.NewResponse(-1, err.Error(), nil); err == nil {
		w.Write(resp)
	}
}

// 删除任务服务
// POST {"jobName": ""} 任务名
func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		jobName string
		oldJob  *common.CronJob
		resp    []byte
	)
	// 1.解析表单数据
	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	// 2.获取表单数据
	jobName = r.PostForm.Get("jobName")
	// 3.etcd中删除任务
	if oldJob, err = G_jobMgr.DeleteJob(jobName); err != nil {
		goto ERR
	}
	// 4. 返回响应数据
	if resp, err = common.NewResponse(1, "success", oldJob); err == nil {
		w.Write(resp)
	}
	return
ERR:
	// 请求异常
	if resp, err = common.NewResponse(-1, err.Error(), nil); err == nil {
		w.Write(resp)
	}
}

// 获取任务列表服务
// GET
func handleJobList(w http.ResponseWriter, r *http.Request) {
	var (
		jobList []*common.CronJob
		err     error
		resp    []byte
	)
	// 1.从etcd当中获取任务列表
	if jobList, err = G_jobMgr.GetJobs(); err != nil {
		goto ERR
	}
	// 2.返回响应数据
	if resp, err = common.NewResponse(1, "success", jobList); err == nil {
		w.Write(resp)
	}
	return
ERR:
	// 请求异常
	if resp, err = common.NewResponse(-1, err.Error(), nil); err == nil {
		w.Write(resp)
	}
}

// 强制杀死某个任务
// POST
func handleJobKill(w http.ResponseWriter, r *http.Request) {
	var (
		jobName string
		err     error
		resp    []byte
	)
	// 1.解析表单数据
	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	// 2.获取任务名
	jobName = r.PostForm.Get("jobName")
	// 3.去etcd添加kill标记
	if err = G_jobMgr.KillJob(jobName); err != nil {
		goto ERR
	}
	// 4.返回响应数据
	if resp, err = common.NewResponse(1, "success", nil); err == nil {
		w.Write(resp)
	}

	return
ERR:
	// 请求异常
	if resp, err = common.NewResponse(-1, err.Error(), nil); err == nil {
		w.Write(resp)
	}
}
