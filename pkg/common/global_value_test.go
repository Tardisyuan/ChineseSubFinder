package common

import (
	"testing"
	"time"
)

func TestAccessTokenLifetime(t *testing.T) {
	now := time.Now()
	timeNow = func() time.Time { return now }
	defer func() { timeNow = time.Now }()

	if ok, _ := CheckAccessToken(""); ok {
		t.Fatal("empty token accepted")
	}
	SetAccessToken("tok")
	if ok, renewed := CheckAccessToken("tok"); !ok || renewed {
		t.Fatal("fresh token: want ok, not renewed")
	}
	if ok, _ := CheckAccessToken("tox"); ok {
		t.Fatal("wrong token accepted")
	}
	// used after 20 days: still valid, lifetime starts again
	now = now.Add(20 * 24 * time.Hour)
	if ok, renewed := CheckAccessToken("tok"); !ok || !renewed {
		t.Fatal("token used within its lifetime: want ok and renewed")
	}
	now = now.Add(20 * 24 * time.Hour)
	if ok, _ := CheckAccessToken("tok"); !ok {
		t.Fatal("refreshed token expired too early")
	}
	// unused for longer than the lifetime: gone
	now = now.Add(AccessTokenLifetime + time.Second)
	if ok, _ := CheckAccessToken("tok"); ok || GetAccessToken() != "" {
		t.Fatal("expired token accepted")
	}
}
