package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/pkg/types"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/dao"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/model"
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
	reader     *MyConsumer
	readerOnce sync.Once
)

type MyConsumer struct {
	r           *kafka.Reader
	orderLogDao dao.OrderLogDao
}

func InitKafka() {
	initKafkaWriter()
	initKafkaReader()
}

func CloseKafka() {
	closeKafkaWriter()
	closeKafkaReader()
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

func initKafkaReader() {
	brokerAddr := fmt.Sprintf("%s:%d", config.Config.KafkaConfig.Host, config.Config.KafkaConfig.Port)
	readerOnce.Do(func() {
		kafkaReader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:   []string{brokerAddr},
			Topic:     "order_status_changed",
			Partition: 0,
			MaxBytes:  10e6,
		})
		reader = &MyConsumer{
			r:           kafkaReader,
			orderLogDao: dao.GetOrderLogDao(),
		}
	})
}

func closeKafkaReader() {
	if reader != nil && reader.r != nil {
		if err := reader.r.Close(); err != nil {
			log.Logger.Errorf("failed to close reader: %s", err.Error())
		}
	}
}

func GetReader() *MyConsumer {
	return reader
}

func (mc *MyConsumer) ConsumeMessage(ctx context.Context) {
	for {
		msgRaw, err := mc.r.ReadMessage(ctx)
		if err != nil {
			log.Logger.Errorf("read message failed, err = %s", err.Error())
			break
		}
		log.Logger.Infof("get message: %s", string(msgRaw.Value))
		var msg types.OrderStatusChangedMessage
		err = JSONDecode(string(msgRaw.Value), &msg)
		if err != nil {
			log.Logger.Errorf("parse json failed, err = %s", err.Error())
			continue
		}
		_, err = mc.orderLogDao.Create(ctx, &model.OrderStatusLog{
			OrderNo: msg.OrderNo,
			UserID: msg.UserId,
			CurrentStatus: msg.CurrentStatus,
			Remark: msg.Remark,
			CreateTime: time.Now(),
		})
		if err != nil {
			log.Logger.Errorf("create order log failed, err = %s", err.Error())
			break
		}
	}
}
