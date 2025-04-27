package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/segmentio/kafka-go"
)

// KafkaClient represents a Kafka client for producing and consuming messages
type KafkaClient struct {
	producers map[string]*kafka.Writer
	consumers map[string]*kafka.Reader
	brokers   []string
	group     string
}

// NewKafkaClient creates a new Kafka client
func NewKafkaClient(cfg *config.KafkaConfig) *KafkaClient {
	return &KafkaClient{
		producers: make(map[string]*kafka.Writer),
		consumers: make(map[string]*kafka.Reader),
		brokers:   cfg.Brokers,
		group:     cfg.ConsumerGroup,
	}
}

// CreateProducer creates a new Kafka producer for a topic
func (k *KafkaClient) CreateProducer(topic string) error {
	if _, exists := k.producers[topic]; exists {
		return nil
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(k.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}

	k.producers[topic] = writer
	return nil
}

// CreateConsumer creates a new Kafka consumer for a topic
func (k *KafkaClient) CreateConsumer(topic string) error {
	if _, exists := k.consumers[topic]; exists {
		return nil
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     k.brokers,
		Topic:       topic,
		GroupID:     k.group,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.FirstOffset,
		MaxWait:     500 * time.Millisecond,
	})

	k.consumers[topic] = reader
	return nil
}

// PublishMessage publishes a message to a Kafka topic
func (k *KafkaClient) PublishMessage(ctx context.Context, topic string, key string, data interface{}) error {
	producer, exists := k.producers[topic]
	if !exists {
		if err := k.CreateProducer(topic); err != nil {
			return err
		}
		producer = k.producers[topic]
	}

	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	err = producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})

	if err != nil {
		return fmt.Errorf("error writing message to Kafka: %w", err)
	}

	return nil
}

// ConsumeMessages consumes messages from a Kafka topic and processes them using a handler function
func (k *KafkaClient) ConsumeMessages(ctx context.Context, topic string, handler func([]byte) error) error {
	consumer, exists := k.consumers[topic]
	if !exists {
		if err := k.CreateConsumer(topic); err != nil {
			return err
		}
		consumer = k.consumers[topic]
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context done, stopping Kafka consumer for topic %s", topic)
			return ctx.Err()
		default:
			msg, err := consumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message from Kafka: %v", err)
				continue
			}

			if err := handler(msg.Value); err != nil {
				log.Printf("Error processing message: %v", err)
				// Continue processing other messages
			}
		}
	}
}

// Close closes all Kafka producers and consumers
func (k *KafkaClient) Close() error {
	for topic, producer := range k.producers {
		if err := producer.Close(); err != nil {
			log.Printf("Error closing producer for topic %s: %v", topic, err)
		}
	}

	for topic, consumer := range k.consumers {
		if err := consumer.Close(); err != nil {
			log.Printf("Error closing consumer for topic %s: %v", topic, err)
		}
	}

	return nil
}