package whatsapp

import "testing"

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"081234567890", "6281234567890"},
		{"6281234567890", "6281234567890"},
		{"+6281234567890", "6281234567890"},
		{"0062 81234567890", "6281234567890"},
		{"0812-3456-7890", "6281234567890"},
	}
	for _, c := range cases {
		got := normalizePhone(c.in)
		if got != c.want {
			t.Errorf("normalizePhone(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidPhoneRejectsTooShortOrTooLong(t *testing.T) {
	if _, err := validPhone("123"); err == nil {
		t.Fatalf("expected error buat nomor terlalu pendek")
	}
	if _, err := validPhone("1234567890123456"); err == nil {
		t.Fatalf("expected error buat nomor terlalu panjang")
	}
	if _, err := validPhone("081234567890"); err != nil {
		t.Fatalf("nomor valid seharusnya lolos: %v", err)
	}
}

func TestE164AddsPlusPrefix(t *testing.T) {
	got, err := e164("081234567890")
	if err != nil {
		t.Fatalf("e164: %v", err)
	}
	if got != "+6281234567890" {
		t.Fatalf("e164 = %q, want +6281234567890", got)
	}
}
