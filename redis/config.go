package redis

import "fmt"

type Config struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}

func (c Config) Addr() string {
	return fmt.Sprintf("%v:%v", c.Host, c.Port)
}
