package kafka

import (
	"encoding/json"
	"testing"
)

func TestWhatsAppMessageEvent_JSONRoundTrip(t *testing.T) {
	original := WhatsAppMessageEvent{
		RecipientPhone: "5543999999999",
		RecipientName:  "Pedrinho",
		Message:        "Olá, Pedrinho, lembrete de mensalidade.",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded WhatsAppMessageEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestWhatsAppMessageEvent_JSONFieldNames(t *testing.T) {
	event := WhatsAppMessageEvent{
		RecipientPhone: "5511999999999",
		RecipientName:  "Ana",
		Message:        "Mensagem de teste",
	}

	data, _ := json.Marshal(event)
	var raw map[string]string
	json.Unmarshal(data, &raw)

	fields := map[string]string{
		"recipient_phone": event.RecipientPhone,
		"recipient_name":  event.RecipientName,
		"message":         event.Message,
	}
	for key, want := range fields {
		if got := raw[key]; got != want {
			t.Errorf("JSON field %q = %q, want %q", key, got, want)
		}
	}
}

func TestWhatsAppMessageEvent_UnmarshalPartial(t *testing.T) {
	// Garante que campos ausentes ficam como zero value (não causam erro)
	data := []byte(`{"recipient_phone":"5511999999999"}`)
	var event WhatsAppMessageEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("unmarshal partial: %v", err)
	}
	if event.RecipientPhone != "5511999999999" {
		t.Errorf("RecipientPhone = %q, want %q", event.RecipientPhone, "5511999999999")
	}
	if event.RecipientName != "" {
		t.Errorf("RecipientName should be empty, got %q", event.RecipientName)
	}
}
