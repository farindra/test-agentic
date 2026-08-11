package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// toJID: bagian user di JID TANPA "+" — "62811459855@s.whatsapp.net".
func toJID(phone string) (types.JID, error) {
	p, err := validPhone(phone)
	if err != nil {
		return types.JID{}, err
	}
	return types.NewJID(p, types.DefaultUserServer), nil
}

func (m *Manager) Send(ctx context.Context, id, phone, text string) error {
	ls := m.get(id)
	if ls == nil || !ls.client.IsLoggedIn() {
		return fmt.Errorf("sesi %q belum tersambung", id)
	}
	jid, err := toJID(phone)
	if err != nil {
		return err
	}
	_, err = ls.client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(text),
	})
	return err
}

// Check: apakah nomor terdaftar di WhatsApp.
func (m *Manager) Check(ctx context.Context, id, phone string) (bool, error) {
	ls := m.get(id)
	if ls == nil || !ls.client.IsLoggedIn() {
		return false, fmt.Errorf("sesi %q belum tersambung", id)
	}
	p, err := e164(phone)
	if err != nil {
		return false, err
	}
	res, err := ls.client.IsOnWhatsApp(ctx, []string{p})
	if err != nil || len(res) == 0 {
		return false, err
	}
	return res[0].IsIn, nil
}
