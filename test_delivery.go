package main

import (
	"context"
	"fmt"
	"log"
	"time"

	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	conn, err := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := messagev1.NewMessageApiClient(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-user-id", "0764724a-714a-4952-b94f-dc29f5505417")
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	r, err := c.SendMessage(ctx, &messagev1.SendMessageRequest{
		SenderUserId:   "0764724a-714a-4952-b94f-dc29f5505417",
		ConversationId: "0764724a-714a-4952-b94f-dc29f5505417",
		Content:        "Final Delivery Test",
		IdempotencyKey: fmt.Sprintf("test-%d", time.Now().Unix()),
		MessageType:    "text",
	})
	if err != nil {
		log.Fatalf("could not send message: %v", err)
	}
	fmt.Printf("Message sent: %s\n", r.GetMessage().GetMessageId())
}
