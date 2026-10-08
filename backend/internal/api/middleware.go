package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"card-system/internal/auth"
	"card-system/internal/db"
	"card-system/internal/model"

	"github.com/gin-gonic/gin"
)

// ---- 管理端 JWT 认证 ----

func (s *Server) adminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录已过期"})
			return
		}
		claims, ok := auth.ParseToken(s.cfg.JWTSecret, strings.TrimPrefix(h, "Bearer "))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已过期，请重新登录"})
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("admin_name", claims.Username)
		c.Next()
	}
}

// ---- App 端签名验证 + 防重放 ----

const signatureWindow = 5 * time.Minute

func (s *Server) signVerify() gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.GetHeader("X-App-Id")
		tsStr := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		sig := c.GetHeader("X-Signature")
		if appID == "" || tsStr == "" || nonce == "" || sig == "" {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeSignError, "缺少签名请求头", nil))
			return
		}
		if appID != s.appID() {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeSignError, "app_id 不匹配", nil))
			return
		}
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeSignError, "时间戳格式错误", nil))
			return
		}
		now := time.Now().UnixMilli()
		diff := now - ts
		if diff < 0 {
			diff = -diff
		}
		if diff > signatureWindow.Milliseconds() {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeSignError, "时间戳偏差超过 5 分钟", nil))
			return
		}
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
		mac := hmac.New(sha256.New, []byte(s.appSecret()))
		mac.Write([]byte(tsStr + nonce + string(body)))
		expect := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(expect)) {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeSignError, "签名校验失败", nil))
			return
		}
		// nonce 防重放：插入失败说明 5 分钟窗口内已出现过
		var exists int
		s.db.QueryRow("SELECT 1 FROM nonces WHERE nonce = ?", nonce).Scan(&exists)
		if exists == 1 {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeReplay, "重复请求", nil))
			return
		}
		if _, err := s.db.Exec("INSERT INTO nonces(nonce, created_at) VALUES(?, ?)", nonce, now); err != nil {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeReplay, "重复请求", nil))
			return
		}
		db.CleanExpiredNonces(s.db, now-signatureWindow.Milliseconds())
		c.Next()
	}
}

// ---- 简单 IP 频控（内存滑动窗口，每 IP 每分钟 60 次） ----

type ipLimiter struct {
	mu sync.Mutex
	m  map[string][]int64
}

func newIPLimiter() *ipLimiter { return &ipLimiter{m: make(map[string][]int64)} }

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().UnixMilli()
	list := l.m[ip]
	kept := list[:0]
	for _, t := range list {
		if now-t < 60_000 {
			kept = append(kept, t)
		}
	}
	if len(kept) >= 60 {
		l.m[ip] = kept
		return false
	}
	l.m[ip] = append(kept, now)
	return true
}

func (s *Server) rateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.limiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusOK, appResp(model.CodeTooMany, "请求过于频繁", nil))
			return
		}
		c.Next()
	}
}

func appResp(code int, msg string, data any) gin.H {
	if msg == "" {
		msg = model.MsgOf(code)
	}
	return gin.H{"code": code, "msg": msg, "data": data}
}

func adminErr(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

func adminOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}
