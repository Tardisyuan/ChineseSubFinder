package lock

import "testing"

func TestLockIsTryLock(t *testing.T) {
	l := NewLock()
	if l.Lock() == false {
		t.Fatal("first Lock() on a fresh lock must succeed")
	}
	if l.Lock() == true {
		t.Fatal("second Lock() must fail instead of blocking")
	}
	l.Unlock()
	if l.Lock() == false {
		t.Fatal("Lock() after Unlock() must succeed")
	}
}
