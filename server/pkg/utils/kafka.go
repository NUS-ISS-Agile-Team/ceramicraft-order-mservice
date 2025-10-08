package utils

import (
	"context"
	"fmt"
	"sync"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/segmentio/kafka-go"
)

type Writer interface {
	SendMsg(ctx context.Context, topic, key, value string) error
}

type MyWriter struct {
	kafkaWriter *kafka.Writer
}

var (
	writer     *MyWriter
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
		kafkaWriter := &kafka.Writer{
			Addr:                   kafka.TCP(brokerAddr),
			Balancer:               &kafka.LeastBytes{},
			RequiredAcks:           kafka.RequireAll,
			Async:                  false,
			AllowAutoTopicCreation: true,
		}
		writer = &MyWriter{
			kafkaWriter: kafkaWriter,
		}
	})
}

func closeKafkaWriter() {
	if writer != nil && writer.kafkaWriter != nil {
		if err := writer.kafkaWriter.Close(); err != nil {
			log.Logger.Errorf("CloseKafkaWriter: failed to close writer, err %s", err.Error())
		}
	}
}

func (myWriter *MyWriter) SendMsg(ctx context.Context, topic, key, value string) error {
	err := myWriter.kafkaWriter.WriteMessages(ctx, kafka.Message{
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

func GetWriter() *MyWriter {
	return writer
}
