package websocket

import (
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	sharedv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/shared/v1"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const (
	SendQueueSize = 128
	writeWait     = 10 * time.Second
	pongWait      = 60 * time.Second
	pingPeriod    = (pongWait * 9) / 10
)

type Session struct {
	ID       string
	UserID   string
	DeviceID string

	Conn      *websocket.Conn
	SendQueue chan []byte
	done      chan struct{}
	closed    atomic.Int32
	ready     atomic.Bool

	resumeBuffer []bufferedEvent
	resumeMu     sync.Mutex
}

type bufferedEvent struct {
	env     *sharedv1.EventEnvelope
	payload []byte
}

func NewSession(id, userID, deviceID string, conn *websocket.Conn) *Session {
	return &Session{
		ID:        id,
		UserID:    userID,
		DeviceID:  deviceID,
		Conn:      conn,
		SendQueue: make(chan []byte, SendQueueSize),
		done:      make(chan struct{}),
	}
}

func (s *Session) Start() {
	go s.writeLoop()
}

func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) SetReady() {
	s.ready.Store(true)
}

func (s *Session) IsReady() bool {
	return s.ready.Load()
}

func (s *Session) Buffer(env *sharedv1.EventEnvelope, payload []byte) bool {
	s.resumeMu.Lock()
	defer s.resumeMu.Unlock()

	if s.ready.Load() {
		return false
	}

	s.resumeBuffer = append(s.resumeBuffer, bufferedEvent{
		env:     env,
		payload: payload,
	})
	return true
}

func (s *Session) FlushBufferSorted() {
	s.resumeMu.Lock()
	defer s.resumeMu.Unlock()

	if s.ready.Load() {
		return
	}

	// Sort buffered events
	sort.Slice(s.resumeBuffer, func(i, j int) bool {
		seqI := getSequence(s.resumeBuffer[i].env)
		seqJ := getSequence(s.resumeBuffer[j].env)

		if seqI != 0 && seqJ != 0 && seqI != seqJ {
			return seqI < seqJ
		}

		// Fallback to OccurredAt
		timeI := s.resumeBuffer[i].env.GetOccurredAt().AsTime()
		timeJ := s.resumeBuffer[j].env.GetOccurredAt().AsTime()
		return timeI.Before(timeJ)
	})

	// Mark ready while holding lock to avoid race with Buffer()
	s.ready.Store(true)

	// Deliver buffered events
	for _, b := range s.resumeBuffer {
		if !s.TrySend(b.payload) {
			log.Printf("session: failed to send buffered event user=%s device=%s", s.UserID, s.DeviceID)
		}
	}

	s.resumeBuffer = nil
}

func getSequence(env *sharedv1.EventEnvelope) int64 {
	if env.GetEventType() != sharedv1.EventType_EVENT_TYPE_MESSAGE_SENT {
		return 0
	}

	var event messagev1.MessageSentEvent
	if err := proto.Unmarshal(env.GetPayload(), &event); err != nil {
		return 0
	}

	return event.GetMessage().GetSequence()
}

func (s *Session) TrySend(msg []byte) bool {
	if s.closed.Load() == 1 {
		return false
	}
	select {
	case s.SendQueue <- msg:
		return true
	default:
		log.Printf("session: backpressure overflow user=%s device=%s - dropping connection", s.UserID, s.DeviceID)
		s.CloseWithReason(websocket.CloseInternalServerErr, "backpressure overflow")
		return false
	}
}

func (s *Session) Close() {
	s.CloseWithReason(websocket.CloseNormalClosure, "server closing")
}

func (s *Session) CloseWithReason(code int, reason string) {
	if !s.closed.CompareAndSwap(0, 1) {
		return
	}

	log.Printf("session: closing user=%s device=%s code=%d reason=%s", s.UserID, s.DeviceID, code, reason)
	close(s.done)

	if s.Conn != nil {
		// Send close message to client
		deadline := time.Now().Add(time.Second)
		_ = s.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), deadline)
		s.Conn.Close()
	}
}

func (s *Session) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.Close()
	}()

	for {
		select {
		case msg, ok := <-s.SendQueue:
			_ = s.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = s.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := s.Conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				log.Printf("session: write error user=%s device=%s: %v", s.UserID, s.DeviceID, err)
				return
			}
		case <-ticker.C:
			_ = s.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("session: ping error user=%s device=%s: %v", s.UserID, s.DeviceID, err)
				return
			}
		case <-s.done:
			return
		}
	}
}
