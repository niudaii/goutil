package kafka

import (
	"context"

	"github.com/IBM/sarama"
	v1 "github.com/niudaii/goutil/constants/v1"
	"github.com/niudaii/goutil/threading"
)

type ConsumerGroupHandler struct {
	topicName string
	handler   HandlerFunc
}

func (h ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error { return nil }

func (h ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h ConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		ctx := context.Background()
		ctx = context.WithValue(ctx, v1.TopicKey, h.topicName)
		threading.GoSafe(func() {
			h.handler(ctx, msg.Value)
		})
		sess.MarkMessage(msg, "")
	}
	return nil
}
