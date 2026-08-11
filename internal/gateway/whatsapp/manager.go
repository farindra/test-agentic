// Package whatsapp membungkus whatsmeow jadi manajer sesi multi-nomor: bikin
// sesi baru, pairing lewat QR, kirim pesan, dan neruskan pesan masuk ke
// package conversation supaya bisa dibales otomatis oleh bot yang di-binding.
package whatsapp

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"test-agentic/internal/conversation"
	"test-agentic/internal/store"

	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	wastore "go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type liveSession struct {
	id     string
	client *whatsmeow.Client
	mu     sync.Mutex
	qr     string
	qrTTL  time.Duration
}

type PairingQR struct {
	Image string
	TTL   time.Duration
}

type Status struct {
	Connected bool   `json:"connected"`
	State     string `json:"state"`
	JID       string `json:"jid,omitempty"`
}

type Manager struct {
	st        *store.Store
	hub       *conversation.Hub
	container *sqlstore.Container
	log       waLog.Logger
	mu        sync.RWMutex
	sessions  map[string]*liveSession
	devName   string
}

// pairMu ngelindungi store.DeviceProps yang GLOBAL se-paket whatsmeow: nama
// device ditulis ke situ sesaat sebelum Connect() dipanggil buat pairing.
var pairMu sync.Mutex

// New nyalain sqlstore.Container di atas *sql.DB yang SAMA dengan DB aplikasi
// (satu file SQLite buat semuanya) — dialect "sqlite3" ngasih tau whatsmeow
// pakai query SQLite, terlepas dari nama driver yang dipakai buat buka db-nya.
func New(sqlDB *sql.DB, st *store.Store, hub *conversation.Hub, log waLog.Logger, deviceName string) (*Manager, error) {
	container := sqlstore.NewWithDB(sqlDB, "sqlite3", log)
	if err := container.Upgrade(context.Background()); err != nil {
		return nil, fmt.Errorf("whatsapp: upgrade skema whatsmeow: %w", err)
	}
	if deviceName == "" {
		deviceName = "Test Agentic Bot"
	}
	return &Manager{
		st:        st,
		hub:       hub,
		container: container,
		log:       log,
		sessions:  make(map[string]*liveSession),
		devName:   deviceName,
	}, nil
}

// setDeviceName nimpa store.DeviceProps.Os — itu yang dipajang WhatsApp
// sebagai nama perangkat. Cuma dibaca pas REGISTRASI (pairing), dan itu
// variabel global se-paket, makanya dikunci pairMu biar pairing dua sesi
// barengan nggak saling timpa nama.
func setDeviceName(name string) {
	wastore.DeviceProps.Os = proto.String(name)
}

func (m *Manager) get(id string) *liveSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

func (m *Manager) put(ls *liveSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[ls.id] = ls
}

func (m *Manager) drop(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

// LoadAndConnect: dipanggil sekali saat startup, reconnect semua sesi yang
// udah pernah paired sebelumnya.
func (m *Manager) LoadAndConnect(ctx context.Context) error {
	rows, err := m.st.ListSessions(ctx, store.KindWhatsApp)
	if err != nil {
		return err
	}
	for _, s := range rows {
		if s.DeviceJID == nil || *s.DeviceJID == "" {
			continue
		}
		jid, err := types.ParseJID(*s.DeviceJID)
		if err != nil {
			continue
		}
		device, err := m.container.GetDevice(ctx, jid)
		if err != nil || device == nil {
			continue
		}
		client := whatsmeow.NewClient(device, m.log)
		ls := &liveSession{id: s.ID, client: client}
		client.AddEventHandler(m.handler(s.ID))
		m.put(ls)
		if err := client.Connect(); err != nil {
			m.log.Warnf("connect %s: %v", s.ID, err)
		}
	}
	return nil
}

// handler: update status sesi di DB + neruskan pesan masuk ke conversation.Hub.
func (m *Manager) handler(id string) func(any) {
	return func(evt any) {
		ctx := context.Background()
		switch v := evt.(type) {
		case *events.Connected, *events.PushNameSetting:
			if ls := m.get(id); ls != nil && ls.client.Store.ID != nil {
				jid := ls.client.Store.ID
				_ = m.st.SetWhatsAppLinked(ctx, id, jid.String(), jid.User, "connected")
			}
		case *events.PairSuccess:
			if ls := m.get(id); ls != nil {
				_ = m.st.SetWhatsAppLinked(ctx, id, v.ID.String(), v.ID.User, "connected")
			}
		case *events.LoggedOut:
			_ = m.st.SetWhatsAppUnlinked(ctx, id, "logged_out")
			if ls := m.get(id); ls != nil {
				ls.mu.Lock()
				ls.qr, ls.qrTTL = "", 0
				ls.mu.Unlock()
				ls.client.Disconnect()
				m.drop(id)
			}
		case *events.Message:
			// Diserahkan ke goroutine terpisah: event handler whatsmeow wajib
			// balik cepat, sedangkan alur balasan bot bisa manggil HTTP ke
			// provider AI yang makan waktu beberapa detik.
			go m.handleIncomingMessage(id, v)
		}
	}
}

// StartPairing: mulai/lanjutkan pairing sesi, balikin QR terkini + TTL-nya.
func (m *Manager) StartPairing(ctx context.Context, id string) (PairingQR, error) {
	if ls := m.get(id); ls != nil {
		if ls.client.Store.ID != nil {
			return PairingQR{}, fmt.Errorf("sesi sudah tersambung")
		}
		ls.mu.Lock()
		qr, ttl := ls.qr, ls.qrTTL
		ls.mu.Unlock()
		if qr != "" {
			return PairingQR{Image: qr, TTL: ttl}, nil
		}
		ls.client.Disconnect()
		m.drop(id)
	}

	pairMu.Lock()
	setDeviceName(m.devName)
	device := m.container.NewDevice()
	client := whatsmeow.NewClient(device, m.log)
	ls := &liveSession{id: id, client: client}
	client.AddEventHandler(m.handler(id))
	m.put(ls)
	_ = m.st.SetSessionStatus(ctx, id, "connecting")

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		pairMu.Unlock()
		return PairingQR{}, fmt.Errorf("qr channel: %w", err)
	}
	if err := client.Connect(); err != nil {
		pairMu.Unlock()
		return PairingQR{}, fmt.Errorf("connect: %w", err)
	}
	pairMu.Unlock()

	first := make(chan PairingQR, 1)
	go func() {
		for evt := range qrChan {
			switch evt.Event {
			case whatsmeow.QRChannelEventCode:
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
				if err != nil {
					continue
				}
				cur := PairingQR{
					Image: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
					TTL:   evt.Timeout,
				}
				ls.mu.Lock()
				ls.qr, ls.qrTTL = cur.Image, cur.TTL
				ls.mu.Unlock()
				_ = m.st.SetSessionStatus(context.Background(), id, "qr")
				select {
				case first <- cur:
				default:
				}
			case "success":
				ls.mu.Lock()
				ls.qr, ls.qrTTL = "", 0
				ls.mu.Unlock()
			}
		}

		ls.mu.Lock()
		ls.qr, ls.qrTTL = "", 0
		ls.mu.Unlock()

		if ls.client.Store.ID == nil {
			_ = m.st.SetSessionStatus(context.Background(), id, "disconnected")
			ls.client.Disconnect()
			m.drop(id)
		}
	}()

	select {
	case qr := <-first:
		return qr, nil
	case <-ctx.Done():
		return PairingQR{}, ctx.Err()
	}
}

func (m *Manager) Status(id string) Status {
	ls := m.get(id)
	if ls == nil {
		return Status{Connected: false, State: "disconnected"}
	}
	st := Status{Connected: ls.client.IsConnected() && ls.client.IsLoggedIn()}
	if ls.client.Store.ID != nil {
		st.JID = ls.client.Store.ID.User
	}
	switch {
	case st.Connected:
		st.State = "connected"
	case ls.client.IsConnected():
		st.State = "connecting"
	default:
		st.State = "disconnected"
	}
	return st
}

func (m *Manager) Disconnect(ctx context.Context, id string) error {
	ls := m.get(id)
	if ls != nil {
		if ls.client.IsLoggedIn() {
			_ = ls.client.Logout(ctx)
		}
		ls.client.Disconnect()
		m.drop(id)
	}
	return m.st.SetSessionStatus(ctx, id, "disconnected")
}
