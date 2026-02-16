package main

import (
	"context"
	"log"

	eventsv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/events/v1"

	"github.com/segmentio/kafka-go"

	"google.golang.org/protobuf/proto"
)

func main() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "message.sent.v1",
		GroupID: "phase1",
	})

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		var event eventsv1.MessageSent
		if err := proto.Unmarshal(msg.Value, &event); err != nil {
			log.Fatal(err)
		}

		log.Printf("received: %+v\n", event)
	}
}
