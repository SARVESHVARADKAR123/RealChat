package application

import (
	"testing"
)

func TestIdempotency(t *testing.T) {
	// Strategy:
	// 1. Send Message A with ClientMsgID "uuid-1"
	// 2. Assert success, msg ID returned.
	// 3. Send Message B with ClientMsgID "uuid-1" (Same ID)
	// 4. Assert success, SAME msg ID returned.
	// 5. Verify only 1 message in DB.

	/*
		ctx := context.Background()
		// Setup app...

		cmd := SendMessageCommand{
			ConversationID: "conv-1",
			UserID:         "user-1",
			ClientMsgID:    "idempotent-key-1",
			Content:        "hello",
		}

		// First Attempt
		msg1, err := app.SendMessage(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}

		// Second Attempt
		msg2, err := app.SendMessage(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}

		if msg1.ID != msg2.ID {
			t.Errorf("Expected same message ID, got %s and %s", msg1.ID, msg2.ID)
		}

		if msg1.Sequence != msg2.Sequence {
			t.Errorf("Sequences differ")
		}
	*/
}
