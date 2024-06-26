package kafkax

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"strings"
)

func (c *Consumer) createTopic(topic string) error {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": strings.Join(c.config.URLs, ","),
	}
	if c.config.Kerberos.KeytabPath != "" {
		_ = configMap.SetKey("security.protocol", c.config.Kerberos.SecurityProtocol)
		_ = configMap.SetKey("sasl.mechanisms", c.config.Kerberos.Mechanisms)
		_ = configMap.SetKey("sasl.kerberos.service.name", c.config.Kerberos.ServiceName)
		_ = configMap.SetKey("sasl.kerberos.principal", c.config.Kerberos.Principal)
		_ = configMap.SetKey("sasl.kerberos.keytab", c.config.Kerberos.KeytabPath)
	}
	adminClient, err := kafka.NewAdminClient(configMap)
	if err != nil {
		return err
	}
	_, err = adminClient.CreateTopics(context.Background(), []kafka.TopicSpecification{{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}})
	return err
}
