package common

import (
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

// 定时任务
type CronJob struct {
	Name     string `json:"name"`     // 任务名
	Command  string `json:"command"`  // shell命令
	CronExpr string `json:"cronExpr"` // cron表达式
}

// 任务调度计划
type JobSchedulerPlan struct {
	CronJob  *CronJob
	Expr     *cronexpr.Expression // 解析好的cron表达式
	NextTime time.Time            // 任务下次调度时间
	Name     string
}

// 任务执行状态
type JobExecuteInfo struct {
	Job      *CronJob  // 任务信息
	PlanTime time.Time // 理论上的调度时间
	RealTime time.Time // 实际的调度时间

}

// HTTP接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

// 变化事件
type JobEvent struct {
	EventType int      // SAVE:1,DELETE:2
	CronJob   *CronJob // 任务信息
}

// 创建任务事件
func NewJobEvent(eventType int, job *CronJob) *JobEvent {
	return &JobEvent{
		EventType: eventType,
		CronJob:   job,
	}
}

// 构造任务执行状态
func NewJobExecuteInfo(plan *JobSchedulerPlan) *JobExecuteInfo {
	return &JobExecuteInfo{
		Job:      plan.CronJob,
		PlanTime: plan.NextTime,
		RealTime: time.Now(),
	}
}

// 应答方法
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

// 反序列化job
func UnmarshalJob(value []byte) (ret *CronJob, err error) {
	var (
		job *CronJob
	)
	job = &CronJob{}
	if err = json.Unmarshal(value, &job); err != nil {
		return
	}
	ret = job
	return
}

// 从etcd的key中提取任务名
// /cron/jobs/job10 -> job10
func ExtractJobName(key string) string {
	return strings.TrimPrefix(key, JOB_SAVE_DIR)
}

// 任务执行结果
type JobExcuteResult struct {
	ExcuteInfo *JobExecuteInfo // 脚本执行信息
	Output     []byte          // 脚本输出
	Err        error           // 脚本错误
	StartTime  time.Time       // 启动时间
	EndTime    time.Time       // 结束时间
}

// 构造任务执行结果
func NewJobExcuteResult(excuteInfo *JobExecuteInfo) *JobExcuteResult {
	return &JobExcuteResult{
		ExcuteInfo: excuteInfo,
		Output:     make([]byte,0),
		Err:        nil,
		StartTime:  time.Now(),
	}
}
