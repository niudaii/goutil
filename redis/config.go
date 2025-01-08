package redis

import "fmt"

type Redis struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}

func (r *Redis) Addr() string {
	return fmt.Sprintf("%v:%v", r.Host, r.Port)
}
