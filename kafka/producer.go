package kafka

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/IBM/sarama"
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
	producer sarama.SyncProducer

	logger *zap.SugaredLogger

	noLogTopics []string
}

func NewProducer(config *ProducerConfig) (p *Producer, err error) {
	saramaConfig := sarama.NewConfig()
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
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Partitioner = sarama.NewRandomPartitioner
	saramaConfig.Producer.Return.Successes = true
	var producer sarama.SyncProducer
	producer, err = sarama.NewSyncProducer(config.URLs, saramaConfig)
	if err != nil {
		return
	}
	return &Producer{
		config:      config,
		producer:    producer,
		logger:      zap.L().Named(v1.KafkaProducerLogger).Sugar(),
		noLogTopics: config.NoLogTopics,
	}, nil
}

func (p *Producer) SendJSON(topic string, obj any) {
	shouldLog := !slice.Contain(p.noLogTopics, topic)
	if shouldLog {
		p.logger.Infof(v1.ClientRequest, topic, jsonutil.MustPretty(obj))
	}
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		p.logger.Errorf(v1.ProducerError, err)
		return
	}
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(jsonBytes),
	}
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Errorf(v1.ProducerError, err)
		return
	}
	if shouldLog {
		p.logger.Infof(v1.ProduceOk, partition, offset, humanize.Bytes(uint64(msg.Value.Length())))
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
