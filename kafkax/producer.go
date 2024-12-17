package kafkax

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/dustin/go-humanize"
	v1 "github.com/niudaii/goutil/constants/v1"
	"github.com/niudaii/goutil/errorx"
	"github.com/niudaii/goutil/jsonutil"
	"github.com/niudaii/goutil/slice"
	"github.com/niudaii/goutil/structs"
	"go.uber.org/zap"
)

type Producer struct {
	config   *ProducerConfig
	producer *kafka.Producer
	events   chan kafka.Event
	logger   *zap.SugaredLogger
}

const (
	retryCount = 10
	retrySleep = 30 * time.Second
)

func NewProducer(config *ProducerConfig) (producer *Producer, err error) {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": strings.Join(config.URLs, ","),
	}
	if config.Kerberos.KeytabPath != "" {
		_ = configMap.SetKey("security.protocol", config.Kerberos.SecurityProtocol)
		_ = configMap.SetKey("sasl.mechanisms", config.Kerberos.Mechanisms)
		_ = configMap.SetKey("sasl.kerberos.service.name", config.Kerberos.ServiceName)
		_ = configMap.SetKey("sasl.kerberos.principal", config.Kerberos.Principal)
		_ = configMap.SetKey("sasl.kerberos.keytab", config.Kerberos.KeytabPath)
	}
	var p *kafka.Producer
	for i := 1; i <= retryCount; i++ {
		p, err = kafka.NewProducer(configMap)
		if err == nil {
			break
		}
		zap.L().Sugar().Errorf("conn to kafka err: %v, retry count: %v/%v", err, i, retryCount)
		time.Sleep(retrySleep)
	}
	if err != nil {
		return
	}
	producer = &Producer{
		config:   config,
		producer: p,
		events:   make(chan kafka.Event, 100), // 带缓冲的 channel
		logger:   zap.L().Named(v1.KafkaProducerLogger).Sugar(),
	}
	go producer.handleEvents() // 启动单个 goroutine 处理所有事件
	return producer, nil
}

func (p *Producer) handleEvents() {
	for e := range p.events {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				p.logger.Errorf(v1.ProducerError, ev.TopicPartition.Error)
			} else {
				shouldLog := !slice.Contain(p.config.NoLogTopics, *ev.TopicPartition.Topic)
				if shouldLog {
					p.logger.Infof(v1.ProduceOk, ev.TopicPartition.Partition, ev.TopicPartition.Offset, humanize.Bytes(uint64(int64(len(ev.Value)))))
				}
			}
		}
	}
}

func (p *Producer) SendJSON(topic string, obj any) {
	shouldLog := !slice.Contain(p.config.NoLogTopics, topic)
	if shouldLog {
		p.logger.Infof(v1.ClientRequest, topic, jsonutil.MustPretty(obj))
	}
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		p.logger.Errorf(v1.RequestError, err)
		return
	}
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonBytes,
	}
	err = p.producer.Produce(msg, p.events)
	if err != nil {
		p.logger.Errorf(v1.RequestError, err)
	}
}

func (p *Producer) SendOk(topic string, data interface{}, msg string) {
	p.SendWithResponse(topic, http.StatusOK, data, msg)
}

func (p *Producer) SendOkWithMessage(topic string, msg string) {
	p.SendWithResponse(topic, http.StatusOK, struct{}{}, msg)
}

func (p *Producer) SendErrorWithMessage(topic string, msg string, err error) {
	if err != nil {
		zap.L().Named("[kafka-producer]").Error(
			msg,
			zap.Error(err),
			zap.Any("stack", string(errorx.GetStack(2, 10))),
		)
	}
	p.SendWithResponse(topic, http.StatusInternalServerError, struct{}{}, msg)
}

func (p *Producer) BadRequestWithMessage(topic string, msg string) {
	p.SendWithResponse(topic, http.StatusBadRequest, struct{}{}, msg)
}

func (p *Producer) SendWithResponse(topic string, code int, data interface{}, msg string) {
	response := structs.Response{
		Code:    code,
		Message: msg,
		Data:    data,
	}
	p.SendJSON(topic, response)
}
