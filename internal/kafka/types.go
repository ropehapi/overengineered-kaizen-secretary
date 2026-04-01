package kafka

// WhatsAppMessageEvent é o payload publicado no topic Kafka para cada mensagem a ser enviada.
type WhatsAppMessageEvent struct {
	RecipientPhone string `json:"recipient_phone"`
	RecipientName  string `json:"recipient_name"`
	Message        string `json:"message"`
}
