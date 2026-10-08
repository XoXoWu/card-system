package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"time"

	"card-system/internal/keygen"
	"card-system/internal/model"

	"github.com/gin-gonic/gin"
)

func hashKey(raw string) string {
	n := keygen.Normalize(raw)
	sum := sha256.Sum256([]byte(n))
	return hex.EncodeToString(sum[:])
}

func effectiveStatus(c *model.Card, now int64) string {
	switch {
	case c.Status == "banned":
		return "banned"
	case c.Status == "revoked":
		return "revoked"
	case c.Status == "unused":
		return "unused"
	case c.ExpireAt > 0 && c.ExpireAt < now:
		return "expired"
	default:
		return "active"
	}
}

type cardRow struct {
	model.Card
	KeyHash string
}

func (s *Server) findCard(rawKey string) *cardRow {
	row := s.db.QueryRow(
		`SELECT id, key_hash, key_tail, plan_id, plan_name, is_test, duration_days, max_devices,
		        status, batch_no, note, activated_at, expire_at, banned_at, revoked_at, created_at
		 FROM cards WHERE key_hash = ?`, hashKey(rawKey))
	var c cardRow
	var isTest int
	err := row.Scan(&c.ID, &c.KeyHash, &c.KeyTail, &c.PlanID, &c.PlanName, &isTest, &c.DurationDays,
		&c.MaxDevices, &c.Status, &c.BatchNo, &c.Note, &c.ActivatedAt, &c.ExpireAt, &c.BannedAt, &c.RevokedAt, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return nil
	}
	c.IsTest = isTest == 1
	return &c
}

func (s *Server) boundDeviceCount(cardID int64) int {
	var n int
	s.db.QueryRow("SELECT COUNT(*) FROM devices WHERE card_id = ? AND unbound_at = 0", cardID).Scan(&n)
	return n
}

func (s *Server) deviceBound(cardID int64, deviceID string) bool {
	var n int
	s.db.QueryRow("SELECT COUNT(*) FROM devices WHERE card_id = ? AND device_id = ? AND unbound_at = 0", cardID, deviceID).Scan(&n)
	return n > 0
}

func daysLeft(expireAt, now int64) int {
	if expireAt <= 0 {
		return 0
	}
	d := expireAt - now
	if d < 0 {
		return 0
	}
	return int(d / 86400)
}

// ---- activate 激活 ----

type activateReq struct {
	CardKey    string `json:"card_key" binding:"required"`
	DeviceID   string `json:"device_id" binding:"required"`
	DeviceInfo string `json:"device_info"`
}

func (s *Server) activate(c *gin.Context) {
	var req activateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp(model.CodeBadRequest, "", nil))
		return
	}
	card := s.findCard(req.CardKey)
	if card == nil {
		c.JSON(http.StatusOK, appResp(model.CodeInvalidCard, "", nil))
		return
	}
	now := time.Now().Unix()
	eff := effectiveStatus(&card.Card, now)
	switch eff {
	case "revoked":
		c.JSON(http.StatusOK, appResp(model.CodeRevoked, "", nil))
		return
	case "banned":
		c.JSON(http.StatusOK, appResp(model.CodeBanned, "", nil))
		return
	case "expired":
		c.JSON(http.StatusOK, appResp(model.CodeExpired, "", nil))
		return
	}

	if s.deviceBound(card.ID, req.DeviceID) {
		// 幂等：同一设备重复激活直接返回成功
		s.db.Exec("UPDATE devices SET last_seen = ?, device_info = ? WHERE card_id = ? AND device_id = ?", now, req.DeviceInfo, card.ID, req.DeviceID)
		c.JSON(http.StatusOK, appResp(model.CodeOK, "", s.cardData(card, now)))
		return
	}

	if s.boundDeviceCount(card.ID) >= card.MaxDevices {
		c.JSON(http.StatusOK, appResp(model.CodeDeviceLimit, "", nil))
		return
	}

	if card.Status == "unused" {
		duration := int64(card.DurationDays) * 86400
		if card.IsTest {
			duration = 24 * 3600 // 测试卡：激活后固定 24 小时
		}
		s.db.Exec("UPDATE cards SET status = 'active', activated_at = ?, expire_at = ? WHERE id = ?", now, now+duration, card.ID)
		card.Status = "active"
		card.ActivatedAt = now
		card.ExpireAt = now + duration
	}
	s.db.Exec("INSERT INTO devices(card_id, device_id, device_info, first_seen, last_seen) VALUES(?,?,?,?,?)",
		card.ID, req.DeviceID, req.DeviceInfo, now, now)
	s.logOp(0, "system", "activate", "****"+card.KeyTail, "")
	c.JSON(http.StatusOK, appResp(model.CodeOK, "", s.cardData(card, now)))
}

func (s *Server) cardData(card *cardRow, now int64) gin.H {
	bound := s.boundDeviceCount(card.ID)
	return gin.H{
		"card_tail":   card.KeyTail,
		"plan":        card.PlanName,
		"is_test":     card.IsTest,
		"expire_at":   card.ExpireAt,
		"days_left":   daysLeft(card.ExpireAt, now),
		"devices":     gin.H{"bound": bound, "max": card.MaxDevices},
		"grace_hours": s.graceHours(),
	}
}

// ---- verify 心跳 ----

type verifyReq struct {
	CardKey  string `json:"card_key" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
}

func (s *Server) verify(c *gin.Context) {
	var req verifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp(model.CodeBadRequest, "", nil))
		return
	}
	card := s.findCard(req.CardKey)
	if card == nil {
		c.JSON(http.StatusOK, appResp(model.CodeInvalidCard, "", nil))
		return
	}
	now := time.Now().Unix()
	switch effectiveStatus(&card.Card, now) {
	case "revoked":
		c.JSON(http.StatusOK, appResp(model.CodeRevoked, "", nil))
		return
	case "banned":
		c.JSON(http.StatusOK, appResp(model.CodeBanned, "", nil))
		return
	case "expired":
		c.JSON(http.StatusOK, appResp(model.CodeExpired, "", s.cardData(card, now)))
		return
	case "unused":
		c.JSON(http.StatusOK, appResp(model.CodeCardUnused, "", nil))
		return
	}
	if !s.deviceBound(card.ID, req.DeviceID) {
		c.JSON(http.StatusOK, appResp(model.CodeUnbound, "", nil))
		return
	}
	s.db.Exec("UPDATE devices SET last_seen = ? WHERE card_id = ? AND device_id = ?", now, card.ID, req.DeviceID)
	c.JSON(http.StatusOK, appResp(model.CodeOK, "", s.cardData(card, now)))
}

// ---- renew 续费 ----

type renewReq struct {
	CurrentCardKey string `json:"current_card_key" binding:"required"`
	NewCardKey     string `json:"new_card_key" binding:"required"`
	DeviceID       string `json:"device_id" binding:"required"`
}

func (s *Server) renew(c *gin.Context) {
	var req renewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp(model.CodeBadRequest, "", nil))
		return
	}
	cur := s.findCard(req.CurrentCardKey)
	if cur == nil {
		c.JSON(http.StatusOK, appResp(model.CodeInvalidCard, "", gin.H{"field": "current_card_key"}))
		return
	}
	now := time.Now().Unix()
	switch effectiveStatus(&cur.Card, now) {
	case "revoked", "unused":
		c.JSON(http.StatusOK, appResp(model.CodeRevoked, "", gin.H{"field": "current_card_key"}))
		return
	case "banned":
		c.JSON(http.StatusOK, appResp(model.CodeBanned, "", gin.H{"field": "current_card_key"}))
		return
	}
	if !s.deviceBound(cur.ID, req.DeviceID) {
		c.JSON(http.StatusOK, appResp(model.CodeUnbound, "", gin.H{"field": "current_card_key"}))
		return
	}

	nk := s.findCard(req.NewCardKey)
	if nk == nil {
		c.JSON(http.StatusOK, appResp(model.CodeInvalidCard, "", gin.H{"field": "new_card_key"}))
		return
	}
	if nk.IsTest {
		// 测试卡不可作为续费卡，PRD 规定返回 revoked
		c.JSON(http.StatusOK, appResp(model.CodeRevoked, "test card cannot renew", gin.H{"field": "new_card_key"}))
		return
	}
	if eff := effectiveStatus(&nk.Card, now); eff != "unused" {
		c.JSON(http.StatusOK, appResp(model.CodeRevoked, "card already used", gin.H{"field": "new_card_key"}))
		return
	}

	base := cur.ExpireAt
	if base < now {
		base = now // 已过期则从当下起算，PRD 续费叠加规则
	}
	newExpire := base + int64(nk.DurationDays)*86400
	if _, err := s.db.Exec("UPDATE cards SET expire_at = ? WHERE id = ?", newExpire, cur.ID); err != nil {
		c.JSON(http.StatusOK, appResp(model.CodeBadRequest, "", nil))
		return
	}
	s.db.Exec("UPDATE cards SET status = 'revoked', revoked_at = ?, note = ? WHERE id = ?",
		now, "[续费消耗] 已叠加至 ****"+cur.KeyTail, nk.ID)
	s.logOp(0, "system", "renew", "****"+cur.KeyTail, "新卡 ****"+nk.KeyTail+" 叠加 "+itoa(nk.DurationDays)+" 天")
	c.JSON(http.StatusOK, appResp(model.CodeOK, "", gin.H{
		"card_tail":  cur.KeyTail,
		"plan":       cur.PlanName,
		"expire_at":  newExpire,
		"days_left":  daysLeft(newExpire, now),
		"devices":    gin.H{"bound": s.boundDeviceCount(cur.ID), "max": cur.MaxDevices},
	}))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
