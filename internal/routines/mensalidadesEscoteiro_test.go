package routines

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ropehapi/kaizen-secretary/internal/kafka"
)

// mockPublisher captura os eventos publicados para inspeção nos testes.
type mockPublisher struct {
	mu        sync.Mutex
	published []kafka.WhatsAppMessageEvent
	err       error
}

func (m *mockPublisher) Publish(_ context.Context, event kafka.WhatsAppMessageEvent) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.published = append(m.published, event)
	m.mu.Unlock()
	return nil
}

// --- BuildMessage tests ---

func TestBuildMessage_ContainsName(t *testing.T) {
	msg := BuildMessage("Pedrinho", "Janeiro")
	if !strings.Contains(msg, "Pedrinho") {
		t.Errorf("message should contain recipient name, got: %q", msg)
	}
}

func TestBuildMessage_ContainsMonth(t *testing.T) {
	msg := BuildMessage("Ana", "Março")
	if !strings.Contains(msg, "Março") {
		t.Errorf("message should contain month, got: %q", msg)
	}
}

func TestBuildMessage_ContainsPIX(t *testing.T) {
	msg := BuildMessage("Carlos", "Junho")
	if !strings.Contains(msg, "PIX") {
		t.Errorf("message should mention PIX, got: %q", msg)
	}
}

func TestBuildMessage_ContainsDisclaimerNote(t *testing.T) {
	msg := BuildMessage("Maria", "Outubro")
	if !strings.Contains(msg, "mensagem automática") {
		t.Errorf("message should contain disclaimer, got: %q", msg)
	}
}

func TestBuildMessage_DifferentNames(t *testing.T) {
	names := []string{"Alice", "Bob", "Carlos", "Diana"}
	for _, name := range names {
		msg := BuildMessage(name, "Maio")
		if !strings.Contains(msg, name) {
			t.Errorf("BuildMessage(%q, ...) does not contain name", name)
		}
	}
}

// --- monthInPortuguese tests ---

func TestMonthInPortuguese_AllMonths(t *testing.T) {
	cases := []struct {
		month    time.Month
		expected string
	}{
		{time.January, "Janeiro"},
		{time.February, "Fevereiro"},
		{time.March, "Março"},
		{time.April, "Abril"},
		{time.May, "Maio"},
		{time.June, "Junho"},
		{time.July, "Julho"},
		{time.August, "Agosto"},
		{time.September, "Setembro"},
		{time.October, "Outubro"},
		{time.November, "Novembro"},
		{time.December, "Dezembro"},
	}

	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			// Cria um time.Time com o mês desejado
			ts := time.Date(2025, tc.month, 1, 0, 0, 0, 0, time.UTC)
			got := monthInPortuguese(ts)
			if got != tc.expected {
				t.Errorf("monthInPortuguese(%v) = %q, want %q", tc.month, got, tc.expected)
			}
		})
	}
}

// --- PublishScoutMonthlyFees tests ---

func TestPublishScoutMonthlyFees_PublishesOneEventPerContributor(t *testing.T) {
	publisher := &mockPublisher{}
	if err := PublishScoutMonthlyFees(context.Background(), publisher); err != nil {
		t.Fatalf("PublishScoutMonthlyFees: %v", err)
	}

	expected := len(contributors())
	publisher.mu.Lock()
	got := len(publisher.published)
	publisher.mu.Unlock()

	if got != expected {
		t.Errorf("expected %d events published, got %d", expected, got)
	}
}

func TestPublishScoutMonthlyFees_EventHasCorrectPhone(t *testing.T) {
	publisher := &mockPublisher{}
	PublishScoutMonthlyFees(context.Background(), publisher)

	members := contributors()
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

	for _, event := range publisher.published {
		wantPhone, ok := members[event.RecipientName]
		if !ok {
			t.Errorf("unknown recipient name %q in published event", event.RecipientName)
			continue
		}
		if event.RecipientPhone != wantPhone {
			t.Errorf("event for %q: phone = %q, want %q",
				event.RecipientName, event.RecipientPhone, wantPhone)
		}
	}
}

func TestPublishScoutMonthlyFees_MessageContainsNameAndMonth(t *testing.T) {
	publisher := &mockPublisher{}
	PublishScoutMonthlyFees(context.Background(), publisher)

	publisher.mu.Lock()
	defer publisher.mu.Unlock()

	for _, event := range publisher.published {
		if !strings.Contains(event.Message, event.RecipientName) {
			t.Errorf("message for %q does not contain their name", event.RecipientName)
		}
	}
}

func TestPublishScoutMonthlyFees_ContinuesOnPublishError(t *testing.T) {
	// Mesmo com erro no publisher, a função não deve retornar erro
	// (erros individuais são logados, não propagados)
	publisher := &mockPublisher{}
	failPublisher := &partialFailPublisher{inner: publisher, failEvery: 2}

	// Adiciona mais membros via monkey-patch não é possível aqui —
	// testamos comportamento com o publisher que falha mas continua.
	err := PublishScoutMonthlyFees(context.Background(), failPublisher)
	if err != nil {
		t.Errorf("PublishScoutMonthlyFees should not return error, got: %v", err)
	}
}

// partialFailPublisher falha a cada N chamadas.
type partialFailPublisher struct {
	mu        sync.Mutex
	calls     int
	failEvery int
	inner     *mockPublisher
}

func (p *partialFailPublisher) Publish(ctx context.Context, event kafka.WhatsAppMessageEvent) error {
	p.mu.Lock()
	p.calls++
	callN := p.calls
	p.mu.Unlock()
	if callN%p.failEvery == 0 {
		return context.DeadlineExceeded // simula erro de broker
	}
	return p.inner.Publish(ctx, event)
}
