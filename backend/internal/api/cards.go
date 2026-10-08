package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"card-system/internal/keygen"
	"card-system/internal/model"

	"github.com/gin-gonic/gin"
)

// ---- 批量生成（普通卡） ----

type generateReq struct {
	PlanID int64  `json:"plan_id" binding:"required"`
	Count  int    `json:"count" binding:"required,min=1,max=5000"`
	Prefix string `json:"prefix"`
	Note   string `json:"note"`
}

func validPrefix(p string) bool {
	if p == "" {
		return true
	}
	if len(p) > 8 {
		return false
	}
	for _, r := range p {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func (s *Server) generate(c *gin.Context) {
	var req generateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminErr(c, http.StatusBadRequest, "数量须在 1–5000 之间")
		return
	}
	prefix := strings.ToUpper(strings.TrimSpace(req.Prefix))
	if !validPrefix(prefix) {
		adminErr(c, http.StatusBadRequest, "前缀须为 0–8 位大写字母数字（不含 I/O/0/1）")
		return
	}
	if prefix == "TEST" {
		adminErr(c, http.StatusBadRequest, "TEST 为测试卡保留前缀，正式批次请使用其他前缀")
		return
	}
	var plan model.Plan
	var enabled int
	err := s.db.QueryRow("SELECT id, name, days, max_devices, price, remark, enabled, created_at, updated_at FROM plans WHERE id = ?", req.PlanID).
		Scan(&plan.ID, &plan.Name, &plan.Days, &plan.MaxDevices, &plan.Price, &plan.Remark, &enabled, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		adminErr(c, http.StatusBadRequest, "套餐不存在")
		return
	}
	if enabled != 1 {
		adminErr(c, http.StatusBadRequest, "套餐已停用，请先启用")
		return
	}

	keys := s.insertCards(req.PlanID, plan.Name, plan.Days, plan.MaxDevices, false, prefix, req.Count, req.Note, "")
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "generate", batchOf(keys), fmt.Sprintf("%s × %d", plan.Name, req.Count))
	adminOK(c, gin.H{"keys": keys, "batch_no": batchOf(keys), "count": len(keys)})
}

// ---- 测试卡发放：固定 TEST 前缀、激活后 24h、1 台设备 ----

type testGenReq struct {
	Count int    `json:"count" binding:"required,min=1,max=10"`
	Note  string `json:"note"`
}

func (s *Server) generateTest(c *gin.Context) {
	var req testGenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminErr(c, http.StatusBadRequest, "测试卡单次只能发放 1–10 张")
		return
	}
	keys := s.insertCards(0, "测试卡 · 24h", 0, 1, true, "TEST", req.Count, req.Note, "")
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "test_issue", batchOf(keys), fmt.Sprintf("× %d %s", req.Count, req.Note))
	adminOK(c, gin.H{"keys": keys, "batch_no": batchOf(keys), "count": len(keys)})
}

func batchOf(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return "BATCH-" + time.Now().Format("20060102") + "-" + fmt.Sprintf("%04d", time.Now().Unix()%10000)
}

// insertCards 生成并入库，返回明文卡密（唯一一次返回）
func (s *Server) insertCards(planID int64, planName string, days, maxDevices int, isTest bool, prefix string, count int, note, batchNo string) []string {
	keys := make([]string, 0, count)
	seen := map[string]bool{}
	now := time.Now().Unix()
	if batchNo == "" {
		batchNo = "BATCH-" + time.Now().Format("20060102") + "-" + fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
	}
	for len(keys) < count {
		k := keygen.Gen(prefix)
		if seen[k] {
			continue
		}
		seen[k] = true
		res, err := s.db.Exec(`INSERT INTO cards(key_hash, key_tail, plan_id, plan_name, is_test, duration_days, max_devices, status, batch_no, note, created_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			hashKey(k), k[len(k)-4:], planID, planName, b2i(isTest), days, maxDevices, "unused", batchNo, note, now)
		if err != nil {
			continue // 与历史卡密碰撞，重新生成
		}
		_ = res
		keys = append(keys, k)
	}
	return keys
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---- 列表（筛选 + 分页） ----

func (s *Server) listCards(c *gin.Context) {
	where := []string{"1=1"}
	args := []any{}
	now := time.Now().Unix()

	if q := strings.TrimSpace(c.Query("keyword")); q != "" {
		// 完整卡密按哈希精确匹配，其余按末 4 位 / 批次号 / 备注模糊搜索
		where = append(where, "(key_hash = ? OR key_tail LIKE ? OR batch_no LIKE ? OR note LIKE ?)")
		like := "%" + q + "%"
		args = append(args, hashKey(q), like, like, like)
	}
	if st := c.Query("status"); st != "" && st != "all" {
		if st == "expired" {
			where = append(where, "status = 'active' AND expire_at > 0 AND expire_at < ?")
			args = append(args, now)
		} else {
			where = append(where, "status = ?")
			args = append(args, st)
		}
	}
	if st := c.Query("type"); st == "test" {
		where = append(where, "is_test = 1")
	} else if st == "normal" {
		where = append(where, "is_test = 0")
	}
	if pid := c.Query("plan_id"); pid != "" && pid != "all" {
		where = append(where, "plan_id = ?")
		args = append(args, pid)
	}
	if b := c.Query("batch"); b != "" {
		where = append(where, "batch_no = ?")
		args = append(args, b)
	}

	cond := strings.Join(where, " AND ")
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE "+cond, args...).Scan(&total)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	rows, err := s.db.Query(`SELECT c.id, c.key_tail, c.plan_id, c.plan_name, c.is_test, c.duration_days, c.max_devices,
	        c.status, c.batch_no, c.note, c.activated_at, c.expire_at, c.banned_at, c.revoked_at, c.created_at,
	        (SELECT COUNT(*) FROM devices d WHERE d.card_id = c.id AND d.unbound_at = 0) AS device_count
		FROM cards c WHERE `+cond+` ORDER BY c.id DESC LIMIT ? OFFSET ?`,
		append(args, size, (page-1)*size)...)
	if err != nil {
		adminErr(c, http.StatusInternalServerError, "查询卡密失败")
		return
	}
	defer rows.Close()

	list := []*model.Card{}
	for rows.Next() {
		var card model.Card
		var isTest int
		if err := rows.Scan(&card.ID, &card.KeyTail, &card.PlanID, &card.PlanName, &isTest, &card.DurationDays, &card.MaxDevices,
			&card.Status, &card.BatchNo, &card.Note, &card.ActivatedAt, &card.ExpireAt, &card.BannedAt, &card.RevokedAt, &card.CreatedAt, &card.DeviceCount); err != nil {
			continue
		}
		card.IsTest = isTest == 1
		if card.Status == "active" && card.ExpireAt > 0 && card.ExpireAt < now {
			card.Status = "expired"
		}
		list = append(list, &card)
	}
	adminOK(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

// ---- 详情 ----

func (s *Server) cardDetail(c *gin.Context) {
	var card model.Card
	var isTest int
	now := time.Now().Unix()
	err := s.db.QueryRow(`SELECT id, key_tail, plan_id, plan_name, is_test, duration_days, max_devices,
	        status, batch_no, note, activated_at, expire_at, banned_at, revoked_at, created_at
		FROM cards WHERE id = ?`, c.Param("id")).
		Scan(&card.ID, &card.KeyTail, &card.PlanID, &card.PlanName, &isTest, &card.DurationDays, &card.MaxDevices,
			&card.Status, &card.BatchNo, &card.Note, &card.ActivatedAt, &card.ExpireAt, &card.BannedAt, &card.RevokedAt, &card.CreatedAt)
	if err != nil {
		adminErr(c, http.StatusNotFound, "卡密不存在")
		return
	}
	card.IsTest = isTest == 1
	if card.Status == "active" && card.ExpireAt > 0 && card.ExpireAt < now {
		card.Status = "expired"
	}
	devices := []*model.Device{}
	rows, _ := s.db.Query("SELECT id, card_id, device_id, device_info, first_seen, last_seen, unbound_at FROM devices WHERE card_id = ? ORDER BY unbound_at = 0 DESC, last_seen DESC", card.ID)
	defer rows.Close()
	for rows.Next() {
		var d model.Device
		if rows.Scan(&d.ID, &d.CardID, &d.DeviceID, &d.DeviceInfo, &d.FirstSeen, &d.LastSeen, &d.UnboundAt) == nil {
			devices = append(devices, &d)
		}
	}
	adminOK(c, gin.H{"card": card, "devices": devices})
}

// ---- 单卡操作：封禁 / 解封 / 作废 / 备注 ----

func (s *Server) banCard(c *gin.Context) {
	now := time.Now().Unix()
	res, err := s.db.Exec("UPDATE cards SET status = 'banned', banned_at = ? WHERE id = ? AND status != 'revoked'", now, c.Param("id"))
	if err != nil || nRows(res) == 0 {
		adminErr(c, http.StatusBadRequest, "封禁失败，卡密不存在或已作废")
		return
	}
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "ban", "id:"+c.Param("id"), "")
	adminOK(c, "ok")
}

func (s *Server) unbanCard(c *gin.Context) {
	res, err := s.db.Exec("UPDATE cards SET status = CASE WHEN activated_at > 0 THEN 'active' ELSE 'unused' END, banned_at = 0 WHERE id = ? AND status = 'banned'", c.Param("id"))
	if err != nil || nRows(res) == 0 {
		adminErr(c, http.StatusBadRequest, "解封失败，卡密未处于封禁状态")
		return
	}
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "unban", "id:"+c.Param("id"), "")
	adminOK(c, "ok")
}

func (s *Server) revokeCard(c *gin.Context) {
	res, err := s.db.Exec("UPDATE cards SET status = 'revoked', revoked_at = ? WHERE id = ? AND status != 'revoked'", time.Now().Unix(), c.Param("id"))
	if err != nil || nRows(res) == 0 {
		adminErr(c, http.StatusBadRequest, "作废失败，卡密不存在或已作废")
		return
	}
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "revoke", "id:"+c.Param("id"), "")
	adminOK(c, "ok")
}

type noteReq struct {
	Note string `json:"note"`
}

func (s *Server) cardNote(c *gin.Context) {
	var req noteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminErr(c, http.StatusBadRequest, "参数错误")
		return
	}
	res, err := s.db.Exec("UPDATE cards SET note = ? WHERE id = ?", req.Note, c.Param("id"))
	if err != nil || nRows(res) == 0 {
		adminErr(c, http.StatusNotFound, "卡密不存在")
		return
	}
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "note", "id:"+c.Param("id"), req.Note)
	adminOK(c, "ok")
}

// ---- 设备解绑 ----

func (s *Server) unbindDevice(c *gin.Context) {
	var cardID, deviceID int64
	err := s.db.QueryRow("SELECT card_id, id FROM devices WHERE id = ?", c.Param("deviceId")).Scan(&cardID, &deviceID)
	if err != nil {
		adminErr(c, http.StatusNotFound, "设备不存在")
		return
	}
	res, err := s.db.Exec("UPDATE devices SET unbound_at = ? WHERE id = ? AND unbound_at = 0", time.Now().Unix(), c.Param("deviceId"))
	if err != nil || nRows(res) == 0 {
		adminErr(c, http.StatusBadRequest, "该设备已是解绑状态")
		return
	}
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "unbind", "id:"+c.Param("id"), "device_id:"+strconv.FormatInt(deviceID, 10))
	adminOK(c, "ok")
}

func nRows(res interface{ RowsAffected() (int64, error) }) int64 {
	n, err := res.RowsAffected()
	if err != nil {
		return 0
	}
	return n
}
