package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	segmentio "github.com/segmentio/kafka-go"
)

// mockWriter captura as mensagens escritas para inspeção nos testes.
type mockWriter struct {
	written []segmentio.Message
	err     error
}

func (m *mockWriter) WriteMessages(_ context.Context, msgs ...segmentio.Message) error {
	m.written = append(m.written, msgs...)
	return m.err
}

func (m *mockWriter) Close() error { return nil }

func TestProducer_Publish_MessageKey(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw}

	event := WhatsAppMessageEvent{
		RecipientPhone: "5543999999999",
		RecipientName:  "Pedrinho",
		Message:        "Lembrete de mensalidade",
	}

	if err := p.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if len(mw.written) != 1 {
		t.Fatalf("expected 1 message written, got %d", len(mw.written))
	}

	// Key deve ser o número do destinatário para garantir ordering
	if string(mw.written[0].Key) != event.RecipientPhone {
		t.Errorf("Key = %q, want %q", string(mw.written[0].Key), event.RecipientPhone)
	}
}

func TestProducer_Publish_MessagePayload(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw}

	event := WhatsAppMessageEvent{
		RecipientPhone: "5511888888888",
		RecipientName:  "Ana",
		Message:        "Olá, Ana!",
	}

	p.Publish(context.Background(), event)

	var got WhatsAppMessageEvent
	if err := json.Unmarshal(mw.written[0].Value, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got != event {
		t.Errorf("payload mismatch: got %+v, want %+v", got, event)
	}
}

func TestProducer_Publish_WriterError(t *testing.T) {
	wantErr := errors.New("broker unavailable")
	mw := &mockWriter{err: wantErr}
	p := &Producer{w: mw}

	err := p.Publish(context.Background(), WhatsAppMessageEvent{RecipientPhone: "5500000000000"})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected writer error, got %v", err)
	}
}

func TestProducer_Publish_MultipleEvents(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw}

	events := []WhatsAppMessageEvent{
		{RecipientPhone: "5511111111111", RecipientName: "A", Message: "msg A"},
		{RecipientPhone: "5522222222222", RecipientName: "B", Message: "msg B"},
		{RecipientPhone: "5533333333333", RecipientName: "C", Message: "msg C"},
	}

	for _, e := range events {
		if err := p.Publish(context.Background(), e); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}

	if len(mw.written) != len(events) {
		t.Errorf("expected %d messages, got %d", len(events), len(mw.written))
	}
	for i, msg := range mw.written {
		if string(msg.Key) != events[i].RecipientPhone {
			t.Errorf("message[%d] key = %q, want %q", i, string(msg.Key), events[i].RecipientPhone)
		}
	}
}

func TestProducer_Close(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw}
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
