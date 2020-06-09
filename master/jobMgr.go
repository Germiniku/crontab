package master

import (
	"context"
	"crontab/common"
	"encoding/json"
	"errors"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"time"
)

// 任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	G_jobMgr *JobMgr
)

// 初始化任务管理器
func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
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
	// 赋值给全局单例对象
	G_jobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}
	return
}

// 保存任务
func (jobMgr *JobMgr) SaveJob(job *common.CronJob) (oldJob *common.CronJob, err error) {
	// 把任务保存到/cron/jobs/jobName ->json
	var (
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldJobObj common.CronJob
	)
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	// 保存到etcd

	if putResp, err = G_jobMgr.kv.Put(context.TODO(), saveEtcdKey(job.Name), string(jobValue), clientv3.WithPrevKV()); err != nil {
		fmt.Println(err)
		return
	}
	// 如果是更新操作，那么返回旧值
	if putResp.PrevKv != nil {
		// 对旧值反序列化
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		// 返回旧值
		oldJob = &oldJobObj
	}
	return
}

// 删除任务
func (jobMgr *JobMgr) DeleteJob(jobName string) (oldJob *common.CronJob, err error) {
	var (
		key            string
		deleteResponse *clientv3.DeleteResponse
		data           []byte
		oldJobObj      common.CronJob
	)
	// 1.获取完整的key
	key = saveEtcdKey(jobName)
	// 2.根据key去etcd中删除任务
	if deleteResponse, err = G_jobMgr.kv.Delete(context.TODO(), key, clientv3.WithPrevKV()); err != nil {
		fmt.Println(err)
		return
	}
	if len(deleteResponse.PrevKvs) != 0 {
		// 删除成功
		data = deleteResponse.PrevKvs[0].Value
		if err = json.Unmarshal(data, &oldJobObj); err != nil {
			fmt.Println(err)
			return
		}
		oldJob = &oldJobObj
		return
	}
	// 删除失败
	return nil, errors.New(fmt.Sprintf("没有%s任务", jobName))
}

// 获取任务列表
func (jobMgr *JobMgr) GetJobs() (jobList []*common.CronJob, err error) {
	var (
		key     string
		getResp *clientv3.GetResponse
		cronJob *common.CronJob
	)
	// 1. 获取key
	key = saveEtcdKey("")
	// 2. etcd中获取以key为前缀的所有值
	if getResp, err = G_jobMgr.kv.Get(context.TODO(), key, clientv3.WithPrefix()); err != nil {
		return
	}
	// 按照任务列表的个数初始化任务列表的切片
	jobList = make([]*common.CronJob, getResp.Count)
	// 遍历获取到所有的kvPair
	for _, kvPair := range getResp.Kvs {
		cronJob = &common.CronJob{}
		// 将key的值赋值到定时任务对象上
		if err = json.Unmarshal(kvPair.Value, cronJob); err != nil {
			err = nil
			continue
		}
		// 将定时任务添加到任务列表当中
		jobList = append(jobList, cronJob)
	}
	return
}

// 获取保存etcd中的key
func saveEtcdKey(jobName string) string {
	var (
		prefixName string
	)
	prefixName = common.JOB_SAVE_DIR
	return prefixName + jobName
}

// 获取杀死任务etcd中的key
func getKillTaskKey(jobName string) string {
	var (
		prefixName string
	)
	prefixName = common.JOB_KILL_DIR
	return prefixName + jobName
}

// 杀死任务
func (jobMgr *JobMgr) KillJob(name string) (err error) {
	var (
		key                string
		leaseGrantResponse *clientv3.LeaseGrantResponse
		leaseID            clientv3.LeaseID
	)
	// 获取key
	key = getKillTaskKey(name)
	// 让worker监听到一次put操作即可，创建租期让其稍后过期
	if leaseGrantResponse, err = G_jobMgr.lease.Grant(context.TODO(), 1); err != nil {
		return
	}
	// 租约ID
	leaseID = leaseGrantResponse.ID
	// 设置kill标记
	if _, err = G_jobMgr.kv.Put(context.TODO(), key, "", clientv3.WithLease(leaseID)); err != nil {
		return
	}
	return
}
