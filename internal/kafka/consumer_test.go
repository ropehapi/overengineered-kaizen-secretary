package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	segmentio "github.com/segmentio/kafka-go"
)

// mockReader simula um kafka.Reader para testes sem broker real.
// Retorna as mensagens da fila e bloqueia no ReadMessage quando esgotam,
// aguardando o ctx ser cancelado.
type mockReader struct {
	mu       sync.Mutex
	messages []segmentio.Message
	idx      int
}

func (m *mockReader) ReadMessage(ctx context.Context) (segmentio.Message, error) {
	m.mu.Lock()
	if m.idx < len(m.messages) {
		msg := m.messages[m.idx]
		m.idx++
		m.mu.Unlock()
		return msg, nil
	}
	m.mu.Unlock()
	<-ctx.Done()
	return segmentio.Message{}, ctx.Err()
}

func (m *mockReader) Close() error { return nil }

// mockDeliverer captura eventos entregues e pode simular erro.
type mockDeliverer struct {
	mu        sync.Mutex
	delivered []WhatsAppMessageEvent
	err       error
}

func (m *mockDeliverer) Deliver(_ context.Context, event WhatsAppMessageEvent) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.delivered = append(m.delivered, event)
	m.mu.Unlock()
	return nil
}

func newTestConsumer(reader messageReader, deliverer Deliverer) *Consumer {
	return &Consumer{reader: reader, deliverer: deliverer, deliveryDelay: 0}
}

func kafkaMsg(t *testing.T, event WhatsAppMessageEvent) segmentio.Message {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal test event: %v", err)
	}
	return segmentio.Message{Key: []byte(event.RecipientPhone), Value: data}
}

// --- ReadLoop tests ---

func TestConsumer_ReadLoop_DeliversEvents(t *testing.T) {
	events := []WhatsAppMessageEvent{
		{RecipientPhone: "5511111111111", RecipientName: "A", Message: "msg A"},
		{RecipientPhone: "5522222222222", RecipientName: "B", Message: "msg B"},
	}

	msgs := make([]segmentio.Message, len(events))
	for i, e := range events {
		msgs[i] = kafkaMsg(t, e)
	}

	reader := &mockReader{messages: msgs}
	deliverer := &mockDeliverer{}
	consumer := newTestConsumer(reader, deliverer)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		consumer.ReadLoop(ctx)
		close(done)
	}()

	// Aguarda entrega de todos os eventos antes de cancelar
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		deliverer.mu.Lock()
		n := len(deliverer.delivered)
		deliverer.mu.Unlock()
		if n == len(events) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	deliverer.mu.Lock()
	got := deliverer.delivered
	deliverer.mu.Unlock()

	if len(got) != len(events) {
		t.Fatalf("expected %d delivered, got %d", len(events), len(got))
	}
	for i, e := range events {
		if got[i] != e {
			t.Errorf("delivered[%d] = %+v, want %+v", i, got[i], e)
		}
	}
}

func TestConsumer_ReadLoop_ExitsOnContextCancel(t *testing.T) {
	reader := &mockReader{} // sem mensagens — bloqueia imediatamente
	consumer := newTestConsumer(reader, &mockDeliverer{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		consumer.ReadLoop(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// OK — ReadLoop saiu após cancelamento
	case <-time.After(time.Second):
		t.Fatal("ReadLoop did not exit after context cancellation")
	}
}

func TestConsumer_ReadLoop_SkipsMalformedMessages(t *testing.T) {
	malformed := segmentio.Message{Key: []byte("5500000000000"), Value: []byte("not-json")}
	valid := kafkaMsg(t, WhatsAppMessageEvent{
		RecipientPhone: "5511111111111",
		RecipientName:  "Ana",
		Message:        "OK",
	})

	reader := &mockReader{messages: []segmentio.Message{malformed, valid}}
	deliverer := &mockDeliverer{}
	consumer := newTestConsumer(reader, deliverer)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		consumer.ReadLoop(ctx)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		deliverer.mu.Lock()
		n := len(deliverer.delivered)
		deliverer.mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	deliverer.mu.Lock()
	n := len(deliverer.delivered)
	deliverer.mu.Unlock()

	if n != 1 {
		t.Errorf("expected 1 delivered (skipping malformed), got %d", n)
	}
}

func TestConsumer_ReadLoop_ContinuesOnDeliveryError(t *testing.T) {
	events := []WhatsAppMessageEvent{
		{RecipientPhone: "5511111111111", RecipientName: "A", Message: "msg A"},
		{RecipientPhone: "5522222222222", RecipientName: "B", Message: "msg B"},
	}
	msgs := []segmentio.Message{kafkaMsg(t, events[0]), kafkaMsg(t, events[1])}

	reader := &mockReader{messages: msgs}
	// Deliverer falha na primeira entrega mas continua nas demais
	callCount := 0
	deliverer := &mockDeliverer{}
	firstFail := &failFirstDeliverer{inner: deliverer}

	consumer := newTestConsumer(reader, firstFail)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		consumer.ReadLoop(ctx)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = callCount
		deliverer.mu.Lock()
		n := len(deliverer.delivered)
		deliverer.mu.Unlock()
		if n == 1 { // apenas o segundo evento chega (primeiro falhou)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	deliverer.mu.Lock()
	n := len(deliverer.delivered)
	deliverer.mu.Unlock()

	if n != 1 {
		t.Errorf("expected 1 successful delivery after first error, got %d", n)
	}
}

// failFirstDeliverer falha na primeira chamada, depois delega ao inner.
type failFirstDeliverer struct {
	mu    sync.Mutex
	calls int
	inner *mockDeliverer
}

func (f *failFirstDeliverer) Deliver(ctx context.Context, event WhatsAppMessageEvent) error {
	f.mu.Lock()
	f.calls++
	callN := f.calls
	f.mu.Unlock()
	if callN == 1 {
		return errors.New("simulated delivery error")
	}
	return f.inner.Deliver(ctx, event)
}

// --- MessagingOfficerDeliverer tests ---

func TestMessagingOfficerDeliverer_Deliver_Success(t *testing.T) {
	var capturedBody []byte
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := newMessagingOfficerDeliverer(server.URL, "test-api-key", "test-session")
	event := WhatsAppMessageEvent{
		RecipientPhone: "5511999999999",
		RecipientName:  "Pedrinho",
		Message:        "Lembrete de mensalidade",
	}

	if err := d.Deliver(context.Background(), event); err != nil {
		t.Fatalf("Deliver: %v", err)
	}

	// Verifica headers
	if capturedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", capturedHeaders.Get("Content-Type"))
	}
	if capturedHeaders.Get("x-api-key") != "test-api-key" {
		t.Errorf("x-api-key = %q", capturedHeaders.Get("x-api-key"))
	}
	if capturedHeaders.Get("x-Session-Id") != "test-session" {
		t.Errorf("x-Session-Id = %q", capturedHeaders.Get("x-Session-Id"))
	}

	// Verifica payload
	var body map[string]string
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if body["number"] != event.RecipientPhone {
		t.Errorf("body.number = %q, want %q", body["number"], event.RecipientPhone)
	}
	if body["message"] != event.Message {
		t.Errorf("body.message = %q, want %q", body["message"], event.Message)
	}
}

func TestMessagingOfficerDeliverer_Deliver_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	d := newMessagingOfficerDeliverer(server.URL, "key", "session")
	err := d.Deliver(context.Background(), WhatsAppMessageEvent{RecipientPhone: "5500000000000"})
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
}

func TestMessagingOfficerDeliverer_Deliver_UnreachableServer(t *testing.T) {
	d := newMessagingOfficerDeliverer("http://127.0.0.1:19999", "key", "session")
	err := d.Deliver(context.Background(), WhatsAppMessageEvent{RecipientPhone: "5500000000000"})
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

func TestMessagingOfficerDeliverer_Deliver_ContextCancelled(t *testing.T) {
	// Servidor que demora — context deve cancelar antes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	d := newMessagingOfficerDeliverer(server.URL, "key", "session")
	err := d.Deliver(ctx, WhatsAppMessageEvent{RecipientPhone: "5500000000000"})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
