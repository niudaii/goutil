package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	v1 "github.com/niudaii/goutil/constants/v1"
	"go.uber.org/zap"
)

type Consumer struct {
	config       *ConsumerConfig
	saramaConfig *sarama.Config

	bindings    []HandlerBinding
	middlewares []MiddlewareFunc

	logger *zap.SugaredLogger

	groups []sarama.ConsumerGroup
}

func NewConsumer(config *ConsumerConfig) (consumer *Consumer) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Net.DialTimeout = 10 * time.Minute
	saramaConfig.Net.ReadTimeout = 10 * time.Minute
	saramaConfig.Net.WriteTimeout = 10 * time.Minute
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Group.Session.Timeout = 10 * time.Minute
	saramaConfig.Consumer.Group.Rebalance.Timeout = 10 * time.Minute
	if config.Kerberos.KeytabPath != "" {
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		saramaConfig.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.GSSAPI.DisablePAFXFAST = true
		saramaConfig.Net.SASL.GSSAPI.KerberosConfigPath = config.Kerberos.KerberosConfigPath
		saramaConfig.Net.SASL.GSSAPI.KeyTabPath = config.Kerberos.KeytabPath
		saramaConfig.Net.SASL.GSSAPI.Realm = config.Kerberos.Realm
		saramaConfig.Net.SASL.GSSAPI.ServiceName = config.Kerberos.ServiceName
		saramaConfig.Net.SASL.GSSAPI.Username = config.Kerberos.Username
		saramaConfig.Net.SASL.GSSAPI.BuildSpn = func(serviceName, host string) string {
			ret := fmt.Sprintf("%s/%s", serviceName, host)
			domain, err := net.LookupAddr(host)
			if err == nil {
				if len(domain) > 0 {
					ret = fmt.Sprintf("%s/%s", serviceName, domain[0])
				}
			}
			return ret
		}
	}
	return &Consumer{
		config:       config,
		saramaConfig: saramaConfig,
		bindings:     []HandlerBinding{},
		middlewares:  []MiddlewareFunc{},
		logger:       zap.L().Named(v1.KafkaCustomerLogger).Sugar(),
		groups:       make([]sarama.ConsumerGroup, 0),
	}
}

// AddMiddleware will add a ServerMiddleware to the list of middlewares to be
func (c *Consumer) AddMiddleware(m MiddlewareFunc) {
	c.middlewares = append(c.middlewares, m)
}

// Bind will add a HandlerBinding to the list of bindings
func (c *Consumer) Bind(bingding HandlerBinding) {
	c.bindings = append(c.bindings, bingding)
}

func (c *Consumer) GetBindings() []HandlerBinding {
	return c.bindings
}

func (c *Consumer) ListenAndServe() {
	c.StartConsume()
}

const (
	retryCount = 10
	retrySleep = 30 * time.Second
)

func (c *Consumer) StartConsume() {
	for _, binding := range c.bindings {
		var group sarama.ConsumerGroup
		var err error
		for i := 1; i <= retryCount; i++ {
			group, err = c.conn(binding.HandlerName)
			if err == nil {
				break
			} else {
				c.logger.Errorf("conn to kafka err: %v, retry count: %v/%v", err, i, retryCount)
				time.Sleep(retrySleep)
			}
		}
		c.groups = append(c.groups, group)
		go c.consume(group, binding)
	}
}

func (c *Consumer) conn(handlerName string) (group sarama.ConsumerGroup, err error) {
	var hasGroup bool
	for _, topic := range c.config.Topics {
		if topic.HandlerName == handlerName {
			hasGroup = topic.HasGroup
			break
		}
	}
	var groupId string
	if hasGroup {
		groupId = fmt.Sprintf("%v-%v", c.config.GroupPrefix, handlerName)
	} else {
		groupId = fmt.Sprintf("%v-%v-%v", c.config.GroupPrefix, handlerName, uuid.New().String())
	}
	group, err = sarama.NewConsumerGroup(c.config.URLs, groupId, c.saramaConfig)
	return
}

func (c *Consumer) consume(group sarama.ConsumerGroup, bingding HandlerBinding) {
	handler := MiddlewareChain(bingding.HandlerFunc, c.middlewares...)
	consumerGroupHandler := ConsumerGroupHandler{
		topicName: bingding.TopicName,
		handler:   handler,
	}
	for i := 1; i <= retryCount; i++ {
		err := group.Consume(context.Background(), []string{bingding.TopicName}, consumerGroupHandler)
		if err != nil {
			c.logger.Errorf("group consume err: %v, retry count: %v/%v", err, i, retryCount)
		}
		time.Sleep(retrySleep)
	}
}

func (c *Consumer) Stop() {
	for _, group := range c.groups {
		err := group.Close()
		if err != nil {
			c.logger.Errorf("group close err: %v", err)
		}
	}
}
