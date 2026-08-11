// Package cryptutil mengenkripsi secret (API key provider, token bot) sebelum
// disimpan ke SQLite, pakai AES-256-GCM dengan key dari ENCRYPTION_KEY.
package cryptutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var ErrNoKey = errors.New("cryptutil: encryption key kosong")

type Box struct {
	gcm cipher.AEAD
}

// New menerima key base64 (hasil `openssl rand -base64 32`). Kalau kosong,
// dibalikin ErrNoKey — pemanggil wajib set ENCRYPTION_KEY sebelum bisa
// simpan/baca secret apa pun.
func New(base64Key string) (*Box, error) {
	if base64Key == "" {
		return nil, ErrNoKey
	}
	raw, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, errors.New("cryptutil: ENCRYPTION_KEY bukan base64 valid")
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, errors.New("cryptutil: ENCRYPTION_KEY harus 16/24/32 byte setelah decode")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{gcm: gcm}, nil
}

// Encrypt balikin string base64: nonce||ciphertext.
func (b *Box) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := b.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (b *Box) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	ns := b.gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("cryptutil: ciphertext terlalu pendek")
	}
	nonce, ct := raw[:ns], raw[ns:]
	plain, err := b.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
