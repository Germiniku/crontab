# Crazycrontab
## 传统定时痛点
1. 机器故障,任务停止调度，甚至配置文件丢失
2. 任务数量多，单机到硬件资源消耗殆尽
3. 需要人工去机器上配置cron，任务执行状态查询麻烦

## 总体功能
- 实现web可视化调度任务
- 实现master-slave分布式架构，master节点无状态，wroker节点可扩展
- 利用etcd watch实现worker节点实时获取任务列表，mongodb做日志存储，记录任务执行结果

### Worker功能
1. 任务同步
2. 任务调度
3. 任务执行
4. 日志保存
