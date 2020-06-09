package common

import "encoding/json"

// 定时任务
type CronJob struct {
	Name     string `json:"name"`     // 任务名
	Command  string `json:"command"`  // shell命令
	CronExpr string `json:"cronExpr"` // cron表达式
}

// {"name":"job1","command":"echo helloworld","cronExpr":"*/5 * * * * * *"}

// HTTP接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

func NewResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	var (
		response Response
	)
	response.Data = data
	response.Msg = msg
	response.Errno = errno
	resp, err = json.Marshal(response)
	return
}
