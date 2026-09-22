package utils

import "testing"

func TestDecryptChaCha20FieldKoreanPlaintext(t *testing.T) {
	t.Parallel()

	plain := "홍길동"
	got, err := DecryptChaCha20Field(plain, "Cupitok-1-Gateway")
	if err != nil {
		t.Fatalf("plaintext Korean should not error: %v", err)
	}
	if got != plain {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestEncryptDecryptChaCha20Korean(t *testing.T) {
	t.Parallel()

	key := "Cupitok-123-Gateway"
	plain := "홍길동"
	enc, err := EncryptChaCha20(plain, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := DecryptChaCha20Field(enc, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestDecryptChaCha20RejectsKoreanAsCiphertext(t *testing.T) {
	t.Parallel()

	_, err := DecryptChaCha20("홍길동", "Cupitok-1-Gateway")
	if err == nil {
		t.Fatal("expected base64 error for Korean plaintext")
	}
}
