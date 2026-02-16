package main

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	eventsv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/events/v1"
	"google.golang.org/protobuf/proto"
)

func main() {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "message.sent.v1",
	})

	event := &eventsv1.MessageSent{
		MessageId:      "msg-1",
		ConversationId: "conv-1",
		SenderId:       "user-1",
		Content:        "hello phase 1",
		Timestamp:      time.Now().Unix(),
	}

	bytes, err := proto.Marshal(event)
	if err != nil {
		log.Fatal(err)
	}

	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(event.ConversationId),
		Value: bytes,
		Time:  time.Now(),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("published")
}
