package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An old config with a plaintext password keeps working and is rewritten as a hash by Save.
func TestPasswordHashMigration(t *testing.T) {
	dir := t.TempDir()
	s := NewSettings(dir)
	old := `{"user_info":{"username":"abcd","password":"123456"}}`
	if err := os.WriteFile(filepath.Join(dir, configName), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.read(); err != nil {
		t.Fatal(err)
	}
	if !s.UserInfo.CheckLogin("abcd", "123456") {
		t.Fatal("plaintext config must still log in before it is saved")
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	onDisk, _ := os.ReadFile(filepath.Join(dir, configName))
	if strings.Contains(string(onDisk), `"123456"`) || !isHashedPassword(s.UserInfo.Password) {
		t.Fatal("password still in plaintext after Save")
	}
	hash := s.UserInfo.Password
	if err := s.Save(); err != nil || s.UserInfo.Password != hash {
		t.Fatal("a hash must not be hashed again")
	}

	re := NewSettings(dir)
	if err := re.read(); err != nil {
		t.Fatal(err)
	}
	if !re.UserInfo.CheckLogin("abcd", "123456") {
		t.Fatal("login failed after migration")
	}
	if re.UserInfo.CheckLogin("abcd", "1234567") || re.UserInfo.CheckLogin("abcx", "123456") ||
		re.UserInfo.CheckLogin("abcd", hash) || (&UserInfo{Username: "a"}).CheckLogin("a", "") {
		t.Fatal("wrong credentials accepted")
	}

	// change password: set plaintext, Save hashes it
	re.UserInfo.Password = "newpass1"
	if err := re.Save(); err != nil {
		t.Fatal(err)
	}
	if !re.UserInfo.CheckPassword("newpass1") || re.UserInfo.CheckPassword("123456") {
		t.Fatal("changed password not applied")
	}
}
