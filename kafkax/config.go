package kafkax

import "time"

type ConsumerConfig struct {
	URLs        []string `yaml:"urls" json:"urls"`
	GroupPrefix string   `yaml:"groupPrefix" json:"groupPrefix"`
	Kerberos    Kerberos `yaml:"kerberos" json:"kerberos"`
	Topics      []Topic  `yaml:"topics" json:"topics"`
}

type ProducerConfig struct {
	URLs        []string `yaml:"urls" json:"urls"`
	Kerberos    Kerberos `yaml:"kerberos" json:"kerberos"`
	NoLogTopics []string `yaml:"noLogTopics" json:"noLogTopics"`
}

type Topic struct {
	TopicName   string `yaml:"topicName" json:"topicName"`
	HandlerName string `yaml:"handlerName" json:"handlerName"`
	HasGroup    bool   `yaml:"hasGroup" json:"hasGroup"`
}

type Kerberos struct {
	SecurityProtocol string `yaml:"securityProtocol" json:"securityProtocol"`
	Mechanisms       string `yaml:"mechanisms" json:"mechanisms"`
	ServiceName      string `yaml:"serviceName" json:"serviceName"`
	Principal        string `yaml:"principal" json:"principal"`
	KeytabPath       string `yaml:"keytabPath" json:"keytabPath"`
}

const (
	defaultSleep   = 3 * time.Second
	defaultTimeout = 1 * time.Second
)
