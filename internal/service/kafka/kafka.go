package kafka

import (
	"context"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

type Service struct {
	producer *kafka.Writer
	consumer *kafka.Reader
}

func New(brokers []string, topic string, groupID string, numPartitions, replicationFactor int) (*Service, error) {
	for _, broker := range brokers {
		if err := createTopic(topic, broker, numPartitions, replicationFactor); err != nil {
			return nil, err
		}
	}

	producer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		CommitInterval: time.Second,
	})

	return &Service{
		producer: producer,
		consumer: consumer,
	}, nil
}

func (s *Service) SendMessage(ctx context.Context, key, value []byte) error {
	err := s.producer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to kafka: %v", err)
	}
	return nil
}

func (s *Service) ReadMessage(ctx context.Context) (key, value []byte, err error) {
	msg, err := s.consumer.ReadMessage(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read message from kafka: %v", err)
	}
	return msg.Key, msg.Value, nil
}

func (s *Service) Close() error {
	if err := s.producer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka producer: %w", err)
	}
	if err := s.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka consumer: %w", err)
	}
	return nil
}

func createTopic(topic, broker string, numPartitions, replicationFactor int) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return fmt.Errorf("failed to connect to kafka broker: %w", err)
	}
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	})
	if err != nil {
		if errors.Is(err, kafka.TopicAlreadyExists) {
			log.Printf("kafka topic '%s' already exists", topic)
			return nil
		}
		return fmt.Errorf("failed to create Kafka topic '%s': %w", topic, err)
	} else {
		log.Printf("kafka topic '%s' created successfully", topic)
	}

	return nil
}
