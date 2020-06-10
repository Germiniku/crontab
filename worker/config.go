package worker

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	EtcdEndpoints   []string `json:"etcdEndpoints"`
	EtcdDialTimeout int      `json:"etcdDialTimeout"`
}

var (
	G_config *Config
)

func InitConfig(filename string) (err error) {
	var (
		data []byte
		conf *Config
	)
	// 1.读取配置文件
	if data, err = ioutil.ReadFile(filename); err != nil {
		return
	}
	// 2.JSON反序列化
	if err = json.Unmarshal(data, &conf); err != nil {
		return
	}
	// 3.赋值给全局config
	G_config = conf
	return
}
