package kafkax

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	v1 "github.com/niudaii/goutil/constants/v1"
	"github.com/niudaii/goutil/threading"
	"go.uber.org/zap"
)

type Consumer struct {
	config      *ConsumerConfig
	bindings    []HandlerBinding
	middlewares []MiddlewareFunc
	logger      *zap.SugaredLogger
	stopChan    chan struct{}
}

func NewConsumer(config *ConsumerConfig) (consumer *Consumer) {
	return &Consumer{
		config:      config,
		bindings:    []HandlerBinding{},
		middlewares: []MiddlewareFunc{},
		logger:      zap.L().Named(v1.KafkaCustomerLogger).Sugar(),
		stopChan:    make(chan struct{}),
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
	c.startConsume()
}

func (c *Consumer) startConsume() {
	for _, binding := range c.bindings {
		consumer, err := c.conn(binding.HandlerName)
		if err != nil {
			c.logger.Infof(v1.InitKafkaConsumerError, err)
			return
		}
		go c.consume(consumer, binding)
	}
}

func (c *Consumer) conn(handlerName string) (consumer *kafka.Consumer, err error) {
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
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": strings.Join(c.config.URLs, ","),
		"group.id":          groupId,
		"auto.offset.reset": "latest",
	}
	if c.config.Kerberos.KeytabPath != "" {
		_ = configMap.SetKey("security.protocol", c.config.Kerberos.SecurityProtocol)
		_ = configMap.SetKey("sasl.mechanisms", c.config.Kerberos.Mechanisms)
		_ = configMap.SetKey("sasl.kerberos.service.name", c.config.Kerberos.ServiceName)
		_ = configMap.SetKey("sasl.kerberos.principal", c.config.Kerberos.Principal)
		_ = configMap.SetKey("sasl.kerberos.keytab", c.config.Kerberos.KeytabPath)
	}
	consumer, err = kafka.NewConsumer(configMap)
	return
}

func (c *Consumer) consume(consumer *kafka.Consumer, bingding HandlerBinding) {
	// subscribeTopics
	err := consumer.Subscribe(bingding.TopicName, nil)
	if err != nil {
		c.logger.Errorf("subscribeTopics err: %v", err)
		return
	}
	defer consumer.Close()
	for {
		select {
		case <-c.stopChan:
			return
		default:
			var msg *kafka.Message
			msg, err = consumer.ReadMessage(defaultTimeout)
			if err == nil {
				handler := MiddlewareChain(bingding.HandlerFunc, c.middlewares...)
				ctx := context.Background()
				ctx = context.WithValue(ctx, v1.TopicKey, bingding.TopicName)
				threading.GoSafe(func() {
					handler(ctx, msg.Value)
				})
			} else {
				if !err.(kafka.Error).IsTimeout() {
					c.logger.Errorf("consume err: %v", err)
				}
				if strings.Contains(err.Error(), "Unknown topic or partition") {
					c.logger.Infof("create topic: %v", bingding.TopicName)
					err = c.createTopic(bingding.TopicName)
					if err != nil {
						c.logger.Errorf("create topic err: %v", err)
					}
				}
				time.Sleep(defaultSleep)
			}
		}
	}
}

func (c *Consumer) Stop() {
	close(c.stopChan)
}
