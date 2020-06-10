package worker

import (
	"crontab/common"
	"github.com/gorhill/cronexpr"
	"time"
)

// 任务调度
type Scheduler struct {
	jobEventChan      chan *common.JobEvent               // etcd任务事件
	jobExcuteResultChan chan *common.JobExcuteResult	// 任务结果队列
	jobPlanTable      map[string]*common.JobSchedulerPlan // 任务计划事件表
	jobExecutingTable map[string]*common.JobExecuteInfo // 任务执行表
}

var (
	G_Scheduler *Scheduler
)

// 初始化调度器
func InitScheduler() (err error) {
	G_Scheduler = &Scheduler{
		jobEventChan: make(chan *common.JobEvent, common.JOB_EVENT_CHAN_LENGTH),
		jobPlanTable: make(map[string]*common.JobSchedulerPlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobExcuteResultChan: make(chan *common.JobExcuteResult,common.JOB_EVENT_CHAN_LENGTH),
	}
	// 启动调度协程
	go G_Scheduler.scheduleLoop()
	return nil
}

// 构造任务调度计划
func newJobSchedulerPlan(job *common.CronJob) (*common.JobSchedulerPlan, error) {
	var (
		expr     *cronexpr.Expression
		nextTime time.Time
		err      error
	)
	// 解析cron表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return nil, err
	}
	// 获取下一跳时间
	nextTime = expr.Next(time.Now())
	return &common.JobSchedulerPlan{
		CronJob:  job,
		Expr:     expr,
		NextTime: nextTime,
		Name: job.Name,
	}, err
}

// 推送任务事件
func (scheduler *Scheduler) pushJobEvent(event *common.JobEvent) {
	scheduler.jobEventChan <- event
}

// 处理任务事件
func (scheduler *Scheduler) HandleJobEvent(jobEvent *common.JobEvent) {
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: // 添加任务事件
		var (
			err              error
			jobSchedulerPlan *common.JobSchedulerPlan
		)
		// 生成任务调度计划
		if jobSchedulerPlan, err = newJobSchedulerPlan(jobEvent.CronJob); err != nil {
			return
		}
		// 修改/添加任务事件计划
		scheduler.jobPlanTable[jobEvent.CronJob.Name] =  jobSchedulerPlan

	case common.JOB_EVENT_DELETE: // 删除任务事件
		var (
			ok bool
		)
		// 判断删除的任务事件是否存在
		if _, ok = scheduler.jobPlanTable[jobEvent.CronJob.Name]; ok {
			// 存在则删除该任务事件
			delete(scheduler.jobPlanTable, jobEvent.CronJob.Name)
		}
	}
}

// 重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulerPlan
		nowTime  time.Time
		nearTime *time.Time
	)
	// 如果任务表中没有任务，则睡眠
	if len(scheduler.jobPlanTable) == 0 {
		time.Sleep(1 * time.Second)
		return
	}
	nowTime = time.Now()
	// 1.遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		// 任务时间早于当前或者任务时间等于当前时间则执行任务
		if jobPlan.NextTime.Before(nowTime) || jobPlan.NextTime.Equal(nowTime) {
			// 尝试执行任务
			scheduler.TryStartJob(jobPlan)
			// 更新下次执行时间
			jobPlan.NextTime = jobPlan.Expr.Next(nowTime)
		}
		// 统计最近一个要过期的任务时间
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}
	// 下次调度间隔(最近的时间-当前时间)
	scheduleAfter = (*nearTime).Sub(nowTime)
	return
}

// 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobSchedulerPlan *common.JobSchedulerPlan) {
	// 调度和执行是两件事
	var (
		jobExcuteInfo *common.JobExecuteInfo
		jobExcuting bool
	)
	// 执行的任务可能运行很久，1分钟会调度60次，但只执行1次

	// 如果当前任务正在执行，跳过本次调度
	if jobExcuteInfo,jobExcuting = scheduler.jobExecutingTable[jobSchedulerPlan.CronJob.Name];jobExcuting {
		return
	}
	// 构建执行状态信息
	jobExcuteInfo = common.NewJobExecuteInfo(jobSchedulerPlan)
	// 保存执行状态
	scheduler.jobExecutingTable[jobSchedulerPlan.Name] = jobExcuteInfo
	// TODO:执行任务
	go G_Excuter.ExcuteJob(jobExcuteInfo)
}

// 调度协程
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		jobResult *common.JobExcuteResult
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
	)
	// 初始化一次
	scheduleAfter = scheduler.TrySchedule()

	// 调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: // 监听任务事件变化
			// 对内存中维护的任务实时增删改查
			scheduler.HandleJobEvent(jobEvent)
		case <-scheduleTimer.C:
		case jobResult = <- scheduler.jobExcuteResultChan: // 监听任务事件结果
			scheduler.HandleJobResult(jobResult)
		}
		scheduleAfter = scheduler.TrySchedule()
		scheduleTimer.Reset(scheduleAfter)
	}
}

// 处理任务结果
func (scheduler *Scheduler)HandleJobResult(jobResult *common.JobExcuteResult){
	// 删除正在执行任务表里的任务信息
	delete(scheduler.jobExecutingTable,jobResult.ExcuteInfo.Job.Name)
	// TODO: 记录日志
}

// 回传任务执行结果
func (scheduler *Scheduler)PushJobExcuteResult(result *common.JobExcuteResult){
	scheduler.jobExcuteResultChan <- result
}