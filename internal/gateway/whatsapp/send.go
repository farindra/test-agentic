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

// Send: dipakai jalur "ketik nomor telepon terus kirim" (API kirim manual,
// tombol "send" di UI) — nomor mentah dari manusia, jadi wajib dinormalisasi
// dulu lewat toJID.
func (m *Manager) Send(ctx context.Context, id, phone, text string) error {
	jid, err := toJID(phone)
	if err != nil {
		return err
	}
	return m.sendToJID(ctx, id, jid, text)
}

// sendToJID ngirim ke JID APA ADANYA, tanpa direkonstruksi dari nomor
// telepon. Dipakai buat bales pesan masuk (auto-reply) — JID pengirim yang
// kita terima dari whatsmeow (evt.Info.Sender) kadang berupa LID (identitas
// privasi WhatsApp) bukan nomor telepon, dan itu WAJIB dipakai apa adanya:
// direkonstruksi ulang jadi "<user>@s.whatsapp.net" bikin WhatsApp gak nemu
// sesi enkripsinya ("no LID found ... from server") dan kiriman gagal.
func (m *Manager) sendToJID(ctx context.Context, id string, jid types.JID, text string) error {
	ls := m.get(id)
	if ls == nil || !ls.client.IsLoggedIn() {
		return fmt.Errorf("sesi %q belum tersambung", id)
	}
	_, err := ls.client.SendMessage(ctx, jid, &waE2E.Message{
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
