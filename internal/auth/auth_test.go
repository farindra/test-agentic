package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("s3cret!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "s3cret!" {
		t.Fatalf("hash sama dengan plaintext")
	}
	if !CheckPassword(hash, "s3cret!") {
		t.Fatalf("CheckPassword harusnya true buat password yang benar")
	}
	if CheckPassword(hash, "salah") {
		t.Fatalf("CheckPassword harusnya false buat password yang salah")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	a := New("test-secret")
	tok, err := a.IssueToken("user-1", "admin")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	claims, err := a.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != "user-1" || claims.Username != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	a1 := New("secret-a")
	a2 := New("secret-b")
	tok, err := a1.IssueToken("user-1", "admin")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if _, err := a2.ParseToken(tok); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken dengan secret beda, got %v", err)
	}
}

func TestParseTokenRejectsGarbage(t *testing.T) {
	a := New("test-secret")
	if _, err := a.ParseToken("not-a-jwt"); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken buat token ngaco, got %v", err)
	}
}
