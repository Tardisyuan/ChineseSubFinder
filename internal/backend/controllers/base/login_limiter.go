package base

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/gin-gonic/gin"
)

const (
	loginFreeFails = 5                // failures in a row before the first lockout
	loginBaseLock  = 30 * time.Second // first lockout, doubles with every further failure
	loginMaxLock   = 5 * time.Minute  // longest lockout
	loginForget    = 15 * time.Minute // failures are forgotten this long after the last one
)

type loginFails struct {
	count        int
	last         time.Time
	blockedUntil time.Time
}

// loginLimiter counts failed password checks per client address and locks an address out
// with a growing delay once it keeps failing. In memory only: a restart clears it.
//
// ponytail: keyed on the TCP peer address, because X-Forwarded-For can be forged to get a
// fresh counter per request. Behind a reverse proxy all clients therefore share one
// counter (lockout is capped at loginMaxLock); switch to c.ClientIP() together with
// engine.SetTrustedProxies if per-client limits behind a proxy are needed.
type loginLimiter struct {
	mu    sync.Mutex
	fails map[string]*loginFails
	now   func() time.Time
}

var loginGuard = &loginLimiter{fails: map[string]*loginFails{}, now: time.Now}

// blockedFor returns how long ip still has to wait, 0 if it may try now.
func (l *loginLimiter) blockedFor(ip string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if f := l.fails[ip]; f != nil {
		if wait := f.blockedUntil.Sub(l.now()); wait > 0 {
			return wait
		}
	}
	return 0
}

func (l *loginLimiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for k, f := range l.fails { // drop stale entries so the map cannot grow without bound
		if now.Sub(f.last) > loginForget {
			delete(l.fails, k)
		}
	}
	f := l.fails[ip]
	if f == nil {
		f = &loginFails{}
		l.fails[ip] = f
	}
	f.count++
	f.last = now
	if over := f.count - loginFreeFails; over >= 0 {
		lock := loginMaxLock
		if over < 8 { // beyond that the doubling is past the cap anyway
			if d := loginBaseLock << uint(over); d < lock {
				lock = d
			}
		}
		f.blockedUntil = now.Add(lock)
	}
}

func (l *loginLimiter) success(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, ip)
}

func tooManyAttempts(c *gin.Context, wait time.Duration) {
	seconds := int(wait/time.Second) + 1
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.JSON(http.StatusTooManyRequests, backend2.ReplyCommon{
		Message: fmt.Sprintf("密码错误次数过多，请 %d 秒后再试", seconds)})
}
