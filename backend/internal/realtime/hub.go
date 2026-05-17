package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
)

// Hub - простой диспетчер Server-Sent Events. Он хранит подключенных клиентов и
// рассылает им JSON-события, не блокируя сборщик данных на медленных браузерах.
type Hub struct {
	mu        sync.RWMutex
	clients   map[chan []byte]struct{}
	join      chan chan []byte
	leave     chan chan []byte
	broadcast chan []byte
}

// NewHub создает hub с буферизованным broadcast-каналом, чтобы короткие пики
// событий не блокировали отправителя.
func NewHub() *Hub {
	return &Hub{
		clients:   make(map[chan []byte]struct{}),
		join:      make(chan chan []byte),
		leave:     make(chan chan []byte),
		broadcast: make(chan []byte, 256),
	}
}

// Run обрабатывает регистрацию клиентов, отключение и широковещательную
// рассылку сообщений до завершения контекста приложения.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ch := <-h.join:
			h.mu.Lock()
			h.clients[ch] = struct{}{}
			h.mu.Unlock()
		case ch := <-h.leave:
			h.mu.Lock()
			if _, ok := h.clients[ch]; ok {
				delete(h.clients, ch)
				close(ch)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for ch := range h.clients {
				select {
				case ch <- msg:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Publish сериализует событие в JSON и ставит его в очередь рассылки. Если
// очередь переполнена, событие пропускается, чтобы не тормозить сбор данных.
func (h *Hub) Publish(eventType string, payload any) {
	data, err := json.Marshal(map[string]any{
		"type":    eventType,
		"payload": payload,
	})
	if err != nil {
		return
	}
	select {
	case h.broadcast <- data:
	default:
	}
}

// ServeHTTP держит открытое SSE-соединение с браузером и пишет каждое событие
// в формате `data: ...`.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, 32)
	h.join <- ch
	defer func() { h.leave <- ch }()

	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(msg)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}
