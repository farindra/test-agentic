package whatsapp

import (
	"fmt"
	"os"
	"strings"
)

const (
	phoneMinDigits = 8
	phoneMaxDigits = 15
)

func defaultCountryCode() string {
	if cc := strings.TrimSpace(os.Getenv("WA_DEFAULT_COUNTRY_CODE")); cc != "" {
		return cc
	}
	return "62"
}

// normalizePhone bikin nomor jadi format internasional tanpa "+" dan tanpa
// "0" di depan — satu-satunya bentuk yang dikenal JID WhatsApp.
//
// Nomor lokal ("0811...") kalau diterusin apa adanya jadi JID yang nggak ada,
// dan whatsmeow nggak mesti balikin error yang jelas — bisa nggantung nyoba
// resolve LID. Normalisasi di sini bukan kosmetik.
func normalizePhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	p := b.String()

	switch {
	case strings.HasPrefix(p, "00"):
		p = p[2:]
	case strings.HasPrefix(p, "0"):
		p = defaultCountryCode() + p[1:]
	}
	return p
}

func validPhone(raw string) (string, error) {
	p := normalizePhone(raw)
	if len(p) < phoneMinDigits || len(p) > phoneMaxDigits {
		return "", fmt.Errorf("nomor %q tidak valid (jadi %q setelah dinormalisasi)", raw, p)
	}
	return p, nil
}

// e164: bentuk PAKAI "+" — khusus buat IsOnWhatsApp, yang secara eksplisit
// minta format ini di dokumentasinya. JID whatsmeow sendiri TIDAK boleh
// pakai "+" — dua API ini gampang ketuker.
func e164(phone string) (string, error) {
	p, err := validPhone(phone)
	if err != nil {
		return "", err
	}
	return "+" + p, nil
}
