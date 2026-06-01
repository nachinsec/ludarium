package secret

import "testing"

func TestRoundTrip(t *testing.T) {
	c, err := New("a-stable-session-secret")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := c.Encrypt("sk-mysecretkey")
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-mysecretkey" {
		t.Fatalf("got %q", got)
	}
}

func TestNonceMakesCiphertextUnique(t *testing.T) {
	c, _ := New("secret")
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Fatal("ciphertext should differ each time (random nonce)")
	}
}

func TestWrongKeyFails(t *testing.T) {
	a, _ := New("secret-A")
	b, _ := New("secret-B")
	enc, _ := a.Encrypt("data")
	if _, err := b.Decrypt(enc); err == nil {
		t.Fatal("decrypt with the wrong key must fail")
	}
}
