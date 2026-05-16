package service

import "go.mau.fi/whatsmeow/types/events"

func HandleNewMessage(s WhatsAppService, userID string, evt *events.Message) {
	s.(*whatsAppService).handleNewMessage(userID, evt)
}

func HandleHistorySync(s WhatsAppService, userID string, evt *events.HistorySync) {
	s.(*whatsAppService).handleHistorySync(userID, evt)
}
