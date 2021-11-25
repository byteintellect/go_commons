package event

import (
	"fmt"
	"github.com/byteintellect/go_commons/entity"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type EventConsumer func(event entity.Event)

type KafkaConsumer struct {
	done             chan bool
	consumer         *kafka.Consumer
	consumerMapping  map[string]EventConsumer
	sigChan          chan os.Signal
	channels         []string
	eventConstructor func() entity.Event
}

func getKafkaConsumerConfigMap(config map[string]interface{}) *kafka.ConfigMap {
	configMap := &kafka.ConfigMap{}
	for key, value := range config {
		err := configMap.SetKey(key, value)
		if err != nil {
			log.Println(fmt.Sprintf("An error %v occurred while setting %v: %v", err, key, value))
		}
	}
	return configMap
}

func NewKafkaConsumer(channels []string, config map[string]interface{}, consumerMapping map[string]EventConsumer) *KafkaConsumer {
	sigChan := make(chan os.Signal, 1)
	consumer, err := kafka.NewConsumer(getKafkaConsumerConfigMap(config))
	done := make(chan bool, 1)
	if err != nil {
		log.Fatalf("An error %v occurred while starting kafka consumer.", err)
	}
	err = consumer.SubscribeTopics(channels, nil)
	if err != nil {
		log.Fatalf("An error %v occurred while subscribing to kafka topics %v.", err, channels)
	}
	return &KafkaConsumer{channels: channels, sigChan: sigChan, done: done, consumer: consumer, consumerMapping: consumerMapping}
}

func (kc *KafkaConsumer) getEvent(eventData []byte) entity.Event {
	event := kc.eventConstructor()
	event.FromByte(eventData)
	return event
}

func (kc *KafkaConsumer) Consume(wg *sync.WaitGroup) {
	signal.Notify(kc.sigChan, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		run := true
		defer wg.Done()
		for run == true {
			select {
			case sig := <-kc.sigChan:
				fmt.Printf("Caught signal %v: terminating\n", sig)
				run = false
			case ev := <-kc.consumer.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					_, _ = fmt.Fprintf(os.Stderr, "%% %v\n", e)
					_ = kc.consumer.Assign(e.Partitions)
				case kafka.RevokedPartitions:
					_, _ = fmt.Fprintf(os.Stderr, "%% %v\n", e)
					_ = kc.consumer.Unassign()
				case *kafka.Message:
					domainEvent := kc.getEvent(e.Value)
					wg.Add(1)
					go func(event entity.Event) {
						defer wg.Done()
						if eventConsumer := kc.consumerMapping[domainEvent.GetEntityType()]; eventConsumer != nil {
							eventConsumer(domainEvent)
						} else {
							log.Println(fmt.Sprintf("Event consumer not found for %v event type", domainEvent.GetEntityType()))
						}
					}(domainEvent)
				case kafka.PartitionEOF:
					fmt.Printf("%% Reached %v\n", e)
				case kafka.Error:
					// Errors should generally be considered as informational, the client will try to automatically recover
					_, _ = fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				}
			}
		}
	}()
}

func (kc *KafkaConsumer) Close() {
	err := kc.consumer.Close()
	if err != nil {
		log.Println(fmt.Sprintf("An error %v occurred while closing kafka consumer.", err))
	}
}
