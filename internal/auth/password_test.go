package auth

import "testing"

func TestPasswordHasherRoundTrip(t *testing.T) {
	h := NewPasswordHasher()
	salt, hash, it, err := h.Hash("secret1")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Verify("secret1", salt, hash, it) {
		t.Fatal("verify failed")
	}
	if h.Verify("wrong", salt, hash, it) {
		t.Fatal("should not verify")
	}
}

func TestSessionCreateAndExpire(t *testing.T) {
	sm := NewSessionManager(0)
	token, err := sm.Create(dummyUser("u1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sm.Get(token); err != nil {
		t.Fatal(err)
	}
	sm.Invalidate(token)
	if _, err := sm.Get(token); err != ErrInvalidToken {
		t.Fatalf("got %v", err)
	}
}
