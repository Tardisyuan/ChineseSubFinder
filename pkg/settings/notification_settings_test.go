package settings

import (
	"os"
	"path/filepath"
	"testing"
)

// A config file written before notification_settings existed loads, with notifications off.
func TestOldConfigLoadsWithoutNotificationSettings(t *testing.T) {
	dir := t.TempDir()
	old := `{"user_info":{"username":"abcd","password":"123456"},"emby_settings":{"enable":true,"address_url":"http://1.2.3.4:8096/"}}`
	if err := os.WriteFile(filepath.Join(dir, configName), []byte(old), 0644); err != nil {
		t.Fatal(err)
	}
	s := NewSettings(dir)
	if err := s.read(); err != nil {
		t.Fatal(err)
	}
	n := s.NotificationSettings
	if n == nil || n.Enable || n.HasChannel() || n.DigestMinutes != 5 || !n.OnSuccess || !n.OnFailure {
		t.Errorf("defaults for an old config: %+v", n)
	}
	if s.UserInfo.Username != "abcd" || !s.EmbySettings.Enable {
		t.Error("the rest of the old config was not kept")
	}
}
