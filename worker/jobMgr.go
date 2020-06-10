package worker

import (
	"context"
	"crontab/common"
	"go.etcd.io/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"

	"time"
)

// 任务管理器
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	G_jobMgr *JobMgr
)

// 初始化任务管理器
func InitJobMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)
	// 创建etcd配置文件
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	// 创建etcd客户端
	if client, err = clientv3.New(config); err != nil {
		return
	}
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)
	// 赋值给全局单例对象
	G_jobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}
	// 启动监听任务
	go G_jobMgr.watchJobs()
	return
}

// 监听任务变化
func (jobMgr *JobMgr) watchJobs() (err error) {
	var (
		getResp            *clientv3.GetResponse
		cronJob            *common.CronJob
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResp          clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
		job                *common.CronJob
	)
	// 1. get一下/cron/jobs/下的所有任务，获取当前集群的revision
	if getResp, err = G_jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}
	// 遍历当前任务
	for _, kvpair := range getResp.Kvs {
		// 反序列化json得到job
		if cronJob, err = common.UnmarshalJob(kvpair.Value); err == nil {
			jobEvent = common.NewJobEvent(common.JOB_EVENT_SAVE, cronJob)
			// 把这个job添加到全局的任务队列里
			G_Scheduler.pushJobEvent(jobEvent)

		}
	}
	// 2. 从该reversion向后监听变化事件
	go func() {
		// 从GET对应的后续版本开始监听变化
		watchStartRevision = getResp.Header.Revision + 1
		// 获取监听/cron/jobs/目录的定时任务
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix(), clientv3.WithRev(watchStartRevision))
		// 处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 任务保存事件
					if cronJob, err = common.UnmarshalJob(watchEvent.Kv.Value); err != nil {
						continue
					}
					// 构造一个更新event，
					jobEvent = common.NewJobEvent(common.JOB_EVENT_SAVE, cronJob)
				case mvccpb.DELETE: // 任务被删除
					// DELETE /cron/jobs/job10

					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					job = &common.CronJob{Name: jobName}
					// 构造一个删除event
					jobEvent = common.NewJobEvent(common.JOB_EVENT_DELETE, job)
				}
				//推送给Schduler
				G_Scheduler.pushJobEvent(jobEvent)

			}
		}
	}()
	return
}

// 创建任务执行锁
func (jobMgr *JobMgr)CreateJobLock(jobName string)(jobLock *JobLock){
	// 创建分布式锁
	jobLock = InitJobLock(jobName,jobMgr.kv,jobMgr.lease)
	return
}