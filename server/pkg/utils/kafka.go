package utils

import (
	"context"
	"fmt"
	"sync"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/segmentio/kafka-go"
)

var (
	writer     *kafka.Writer
	writerOnce sync.Once
)

func InitKafka() {
	initKafkaWriter()
}

func CloseKafka() {
	closeKafkaWriter()
}

func initKafkaWriter() {
	writerOnce.Do(func() {
		brokerAddr := fmt.Sprintf("%s:%d", config.Config.KafkaConfig.Host, config.Config.KafkaConfig.Port)
		writer = &kafka.Writer{
			Addr:                   kafka.TCP(brokerAddr),
			Balancer:               &kafka.LeastBytes{},
			RequiredAcks:           kafka.RequireAll,
			Async:                  false,
			AllowAutoTopicCreation: true,
		}
	})
}

func closeKafkaWriter() {
	if writer != nil {
		if err := writer.Close(); err != nil {
			log.Logger.Errorf("CloseKafkaWriter: failed to close writer, err %s", err.Error())
		}
	}
}

func SendMsg(ctx context.Context, topic, key, value string) error {
	err := writer.WriteMessages(ctx, kafka.Message{
		Topic: topic, // 这里可以覆盖默认 topic
		Key:   []byte(key),
		Value: []byte(value),
	})
	if err != nil {
		log.Logger.Errorf("SendMsg: failed, err %s", err.Error())
		return err
	}
	return nil
}
