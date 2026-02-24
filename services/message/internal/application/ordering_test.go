package application

import (
	"testing"
)

// TestConcurrentOrdering is a placeholder for checking strict ordering.
// Real execution requires a running DB instance.
func TestConcurrentOrdering(t *testing.T) {
	// Strategy:
	// 1. Create a conversation.
	// 2. Launch 50 goroutines.
	// 3. Each sends 10 messages.
	// 4. Record returned Sequence numbers.
	// 5. Verify:
	//    - Total messages = 500
	//    - Sequences are unique
	//    - Sequences are 1..500 without gaps

	/*
		ctx := context.Background()
		// Setup app with real DB...

		convID := "ordering-test-conv"
		// CreateConversation...

		var wg sync.WaitGroup
		results := make(chan int64, 500)

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					cmd := SendMessageCommand{
						ConversationID: convID,
						UserID:         fmt.Sprintf("user-%d", workerID),
						ClientMsgID:    uuid.NewString(), // Unique for ordering test
						Content:        "test",
					}
					msg, err := app.SendMessage(ctx, cmd)
					if err != nil {
						t.Error(err)
						return
					}
					results <- msg.Sequence
				}
			}(i)
		}

		wg.Wait()
		close(results)

		// Verification
		seqMap := make(map[int64]bool)
		var maxSeq int64
		for seq := range results {
			if seqMap[seq] {
				t.Errorf("Duplicate sequence: %d", seq)
			}
			seqMap[seq] = true
			if seq > maxSeq {
				maxSeq = seq
			}
		}

		if len(seqMap) != 500 {
			t.Errorf("Expected 500 messages, got %d", len(seqMap))
		}

		for i := int64(1); i <= 500; i++ {
			if !seqMap[i] {
				t.Errorf("Missing sequence: %d", i)
			}
		}
	*/
}
