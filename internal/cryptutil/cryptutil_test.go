package cryptutil

import "testing"

func testKey() string {
	// 32 byte nol, base64-encoded — cukup buat test, jangan dipakai di produksi.
	return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	box, err := New(testKey())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain := "sk-super-secret-api-key"
	ct, err := box.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ct == plain {
		t.Fatalf("ciphertext sama dengan plaintext, harusnya beda")
	}
	got, err := box.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("roundtrip mismatch: got %q want %q", got, plain)
	}
}

func TestDecryptEmptyString(t *testing.T) {
	box, err := New(testKey())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := box.Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestNewWithoutKey(t *testing.T) {
	if _, err := New(""); err != ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

func TestTwoEncryptionsAreDifferent(t *testing.T) {
	box, _ := New(testKey())
	a, _ := box.Encrypt("same-plaintext")
	b, _ := box.Encrypt("same-plaintext")
	if a == b {
		t.Fatalf("dua enkripsi plaintext sama harusnya beda (nonce random), tapi ciphertext identik")
	}
}
