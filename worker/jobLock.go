package worker

import (
	"context"
	"crontab/common"
	"go.etcd.io/etcd/clientv3"
)

// 分布式锁(TXN事务)
type JobLock struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	jobName    string             // 任务名
	CancelFunc context.CancelFunc //中止自动续租
	leaseID    clientv3.LeaseID   // 租约ID
	isLocked   bool               // 是否上锁成功
}

// 初始化分布式锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) *JobLock {
	return &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
		isLocked: false,
	}
}

// 尝试上锁
func (jobLock *JobLock) TryLock() (err error) {
	var (
		txn                clientv3.Txn
		leaseGrantResp     *clientv3.LeaseGrantResponse
		lockKey            string
		cancelCtx          context.Context
		cancelFunc         context.CancelFunc
		leaseId            clientv3.LeaseID
		leaseKeepAliveResp <-chan *clientv3.LeaseKeepAliveResponse
		leasseKeepResp     *clientv3.LeaseKeepAliveResponse
		txnResp            *clientv3.TxnResponse
	)
	// 1.创建租约(5秒)
	if leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 1); err != nil {
		return
	}
	leaseId = leaseGrantResp.ID
	// cancelFunc用于取消自动续租
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())
	jobLock.CancelFunc = cancelFunc
	// 2.自动续租
	if leaseKeepAliveResp, err = jobLock.lease.KeepAlive(cancelCtx, leaseId); err != nil {
		goto FAIL
	}
	go func() {
		for {
			select {
			case leasseKeepResp = <-leaseKeepAliveResp:
				if leasseKeepResp == nil {
					// 租约已经失效
					goto END

				}
			}
		}
	END:
	}()

	// 3.创建事务txn
	txn = jobLock.kv.Txn(context.TODO())
	// 锁路径
	lockKey = common.JOB_LOCK_DIR + jobLock.jobName
	// 4.事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))
	// 5.成功返回 失败释放租约
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}
	// 锁被占用
	if !txnResp.Succeeded {
		err = common.LOCK_ALREADY_REQUIRE
		goto FAIL
	}
	// 上锁成功
	jobLock.leaseID = leaseId
	jobLock.isLocked = true
	return
FAIL:
	cancelFunc()                                  // 取消自动续租
	jobLock.lease.Revoke(context.TODO(), leaseId) // 释放租约
	return
}

// 释放锁
func (jobLock *JobLock) UnLock() {
	if jobLock.isLocked{
		jobLock.CancelFunc()
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseID)
	}
}
