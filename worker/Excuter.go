package worker

import (
	"os/exec"
	"context"
	"crontab/common"
	"time"
)

// 任务执行器
type Excuter struct {
}

var (
	G_Excuter *Excuter
)

// 执行job
func (excuter *Excuter) ExcuteJob(jobExecuteInfo *common.JobExecuteInfo) {
	var (
		cmd             *exec.Cmd
		output          []byte
		err             error
		jobExcuteResult *common.JobExcuteResult
		jobLock         *JobLock
	)
	go func() {
		// 创建任务执行结果
		jobExcuteResult = common.NewJobExcuteResult(jobExecuteInfo)
		// 初始化分布式锁
		jobLock = G_jobMgr.CreateJobLock(jobExecuteInfo.Job.Name)
		err = jobLock.TryLock()
		defer jobLock.UnLock()
		// 上锁失败
		if err != nil {
			jobExcuteResult.Err = err
			jobExcuteResult.EndTime = time.Now()
		} else {
			jobExcuteResult.StartTime = time.Now()
			// 抢锁成功则执行shell
			cmd = exec.CommandContext(context.TODO(), "/bin/bash", "-c", jobExecuteInfo.Job.Command)
			output, err = cmd.CombinedOutput()
			// 记录输出、错误、结束时间
			jobExcuteResult.Output = output
			jobExcuteResult.Err = err
			jobExcuteResult.EndTime = time.Now()
		}
		// 任务执行结束，将结果传递给scheduler，scheduler从excutingInfoTable中删除
		G_Scheduler.PushJobExcuteResult(jobExcuteResult)
	}()
}

func InitExcuter() (err error) {
	G_Excuter = &Excuter{}
	return
}
