package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

// 统计口径遵循 PRD：测试卡不计入销售 KPI 与激活率；过期为动态计算；封禁卡不计入使用中
func (s *Server) statsOverview(c *gin.Context) {
	now := time.Now().Unix()
	var totalCards, activated, inUse, todayActivated, onlineDevices, testIssued, banned int
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE is_test = 0").Scan(&totalCards)
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE is_test = 0 AND activated_at > 0 AND status != 'revoked'").Scan(&activated)
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE is_test = 0 AND status = 'active' AND expire_at > ?", now).Scan(&inUse)
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE is_test = 0 AND activated_at >= ?", todayStart(now)).Scan(&todayActivated)
	s.db.QueryRow("SELECT COUNT(*) FROM devices d JOIN cards c ON c.id = d.card_id WHERE d.unbound_at = 0 AND d.last_seen >= ?", now-900).Scan(&onlineDevices)
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE is_test = 1").Scan(&testIssued)
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE status = 'banned'").Scan(&banned)

	adminOK(c, gin.H{
		"total_cards":     totalCards,
		"activated":       activated,
		"activation_rate": rate(activated, totalCards),
		"in_use":          inUse,
		"today_activated": todayActivated,
		"online_devices":  onlineDevices,
		"test_issued":     testIssued,
		"banned":          banned,
	})
}

func todayStart(now int64) int64 {
	t := time.Unix(now, 0)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
}

func rate(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// 近 30 天每日激活趋势（测试卡除外）
func (s *Server) statsTrend(c *gin.Context) {
	days := 30
	// 用本地日界：按 activated_at 的自然日分组
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1)).Unix()
	rows, err := s.db.Query(`SELECT activated_at / 86400, COUNT(*) FROM cards
		WHERE is_test = 0 AND activated_at >= ? AND status != 'revoked' GROUP BY activated_at / 86400`, start)
	if err != nil {
		adminErr(c, 500, "查询趋势失败")
		return
	}
	defer rows.Close()
	m := map[int64]int{}
	for rows.Next() {
		var day, n int64
		rows.Scan(&day, &n)
		m[day] = int(n)
	}
	list := []gin.H{}
	for i := 0; i < days; i++ {
		d := time.Unix(start, 0).AddDate(0, 0, i)
		list = append(list, gin.H{
			"date": d.Format("01-02"),
			"count": m[d.Unix()/86400],
		})
	}
	adminOK(c, list)
}

// 套餐分布：各套餐的生成量与使用中数量（测试卡除外）
func (s *Server) statsPlanDist(c *gin.Context) {
	now := time.Now().Unix()
	rows, err := s.db.Query(`SELECT plan_name, COUNT(*),
		SUM(CASE WHEN status = 'active' AND expire_at > ? THEN 1 ELSE 0 END)
		FROM cards WHERE is_test = 0 GROUP BY plan_name ORDER BY COUNT(*) DESC`, now)
	if err != nil {
		adminErr(c, 500, "查询分布失败")
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var name string
		var total, inUse int
		rows.Scan(&name, &total, &inUse)
		list = append(list, gin.H{"name": name, "total": total, "in_use": inUse})
	}
	adminOK(c, list)
}

func (s *Server) listOpLogs(c *gin.Context) {
	page, size := pageParams(c)
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM op_logs").Scan(&total)
	rows, err := s.db.Query("SELECT id, admin_id, admin_name, action, target, detail, created_at FROM op_logs ORDER BY id DESC LIMIT ? OFFSET ?", size, (page-1)*size)
	if err != nil {
		adminErr(c, 500, "查询日志失败")
		return
	}
	defer rows.Close()
	type logRow struct {
		ID        int64  `json:"id"`
		AdminID   int64  `json:"admin_id"`
		AdminName string `json:"admin_name"`
		Action    string `json:"action"`
		Target    string `json:"target"`
		Detail    string `json:"detail"`
		CreatedAt int64  `json:"created_at"`
	}
	list := []logRow{}
	for rows.Next() {
		var l logRow
		rows.Scan(&l.ID, &l.AdminID, &l.AdminName, &l.Action, &l.Target, &l.Detail, &l.CreatedAt)
		list = append(list, l)
	}
	adminOK(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}
