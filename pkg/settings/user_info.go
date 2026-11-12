package settings

import (
	"crypto/subtle"

	"golang.org/x/crypto/bcrypt"
)

type UserInfo struct {
	Username string `json:"username" binding:"required,alphanum"`     // 用户名
	Password string `json:"password" binding:"required,min=6,max=30"` // 密码，配置文件里保存的是 bcrypt 哈希
}

func NewUserInfo(userName, password string) *UserInfo {
	return &UserInfo{
		Username: userName,
		Password: password,
	}
}

// isHashedPassword reports whether p is a bcrypt hash. A plaintext password can never look
// like one: the hash is 60 characters, passwords are limited to 30.
func isHashedPassword(p string) bool {
	_, err := bcrypt.Cost([]byte(p))
	return err == nil
}

// hashPassword replaces a plaintext password with its bcrypt hash. Save calls it, so
// every path that stores a password (setup, change password, the save at startup that
// migrates an old plaintext config) ends up with a hash on disk and in memory.
func (u *UserInfo) hashPassword() error {
	if u.Password == "" || isHashedPassword(u.Password) {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword compares a login attempt with the stored password. A stored plaintext
// password (config edited by hand and not saved yet) is still accepted.
func (u *UserInfo) CheckPassword(plain string) bool {
	if u.Password == "" {
		return false
	}
	if isHashedPassword(u.Password) {
		return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(u.Password), []byte(plain)) == 1
}

// CheckLogin checks username and password; the password is checked even when the
// username is wrong, so the response time does not tell which of the two was wrong.
func (u *UserInfo) CheckLogin(username, plain string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(u.Username), []byte(username)) == 1
	return u.CheckPassword(plain) && userOK
}
