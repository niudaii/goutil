package kafka

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

type Kerberos struct {
	Mechanism          string `yaml:"mechanism" json:"mechanism"`
	KeytabPath         string `yaml:"keytabPath" json:"keytabPath"`
	KerberosConfigPath string `yaml:"kerberosConfigPath" json:"kerberosConfigPath"`
	Realm              string `yaml:"realm" json:"realm"`
	ServiceName        string `yaml:"serviceName" json:"serviceName"`
	Username           string `yaml:"username" json:"username"`
}

type Topic struct {
	TopicName   string `yaml:"topicName" json:"topicName"`
	HandlerName string `yaml:"handlerName" json:"handlerName"`
	HasGroup    bool   `yaml:"hasGroup" json:"hasGroup"`
}
