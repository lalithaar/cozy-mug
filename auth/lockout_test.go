package auth

import (
	"testing"
	"time"
)

func TestLockoutLocksAfterThreshold(t *testing.T) {
	l := NewLockout(2, time.Hour, time.Hour)
	key := "alice"

	if !l.Allow(key) {
		t.Fatal("fresh key is locked")
	}
	if locked, _ := l.Fail(key); locked {
		t.Fatal("first failure locked")
	}
	if locked, _ := l.Fail(key); !locked {
		t.Fatal("threshold failure did not lock")
	}
	if l.Allow(key) {
		t.Fatal("locked key is allowed")
	}

	l.Success(key)
	if !l.Allow(key) {
		t.Fatal("success did not clear lock")
	}
}

func TestLockoutWindowResetsOldFailures(t *testing.T) {
	l := NewLockout(2, 10*time.Millisecond, time.Hour)
	key := "bob"
	_, _ = l.Fail(key)

	// First failure ages out of the window, so the next two behave as two fresh
	// attempts: one tolerated, then the threshold is reached on the third.
	time.Sleep(20 * time.Millisecond)
	if locked, _ := l.Fail(key); locked {
		t.Fatal("single fresh failure locked")
	}
	if locked, _ := l.Fail(key); !locked {
		t.Fatal("threshold failure did not lock")
	}
}

func TestChangePassword(t *testing.T) {
	hash, err := HashPassword("old password")
	if err != nil {
		t.Fatal(err)
	}
	next, err := ChangePassword(hash, "old password", "new password")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(next, "new password") {
		t.Fatal("changed password does not verify")
	}
	if _, err := ChangePassword(hash, "wrong", "new password"); err != ErrInvalidPassword {
		t.Fatalf("wrong current password: %v", err)
	}
}
