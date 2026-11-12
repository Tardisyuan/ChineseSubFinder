package base

import (
	"testing"
	"time"
)

func TestLoginLimiter(t *testing.T) {
	now := time.Now()
	l := &loginLimiter{fails: map[string]*loginFails{}, now: func() time.Time { return now }}

	for i := 0; i < loginFreeFails-1; i++ {
		l.fail("a")
	}
	if l.blockedFor("a") != 0 {
		t.Fatal("blocked before the free attempts are used up")
	}
	l.fail("a")
	if l.blockedFor("a") != loginBaseLock || l.blockedFor("b") != 0 {
		t.Fatal("want first lockout for a only")
	}
	now = now.Add(loginBaseLock)
	if l.blockedFor("a") != 0 {
		t.Fatal("lockout did not end")
	}
	l.fail("a")
	if l.blockedFor("a") != 2*loginBaseLock {
		t.Fatal("lockout did not double")
	}
	for i := 0; i < 100; i++ {
		l.fail("a")
	}
	if l.blockedFor("a") != loginMaxLock {
		t.Fatal("lockout not capped")
	}
	l.success("a")
	if l.blockedFor("a") != 0 {
		t.Fatal("success did not reset")
	}
	// stale entries are dropped
	l.fail("c")
	now = now.Add(loginForget + time.Second)
	l.fail("d")
	if _, ok := l.fails["c"]; ok {
		t.Fatal("stale entry kept")
	}
}
