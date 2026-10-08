package api

import (
	"net/http"
	"time"

	"card-system/internal/auth"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminErr(c, http.StatusBadRequest, "请输入用户名与密码")
		return
	}
	var id int64
	var hash string
	err := s.db.QueryRow("SELECT id, password_hash FROM admins WHERE username = ?", req.Username).Scan(&id, &hash)
	if err != nil || !auth.CheckPassword(hash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, err := auth.IssueToken(s.cfg.JWTSecret, id, req.Username, s.cfg.TokenTTL)
	if err != nil {
		adminErr(c, http.StatusInternalServerError, "签发令牌失败")
		return
	}
	s.logOp(id, req.Username, "login", "", "")
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token, "expires_in": int(s.cfg.TokenTTL.Seconds())}})
}

type changePwReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (s *Server) changePassword(c *gin.Context) {
	var req changePwReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminErr(c, http.StatusBadRequest, "新密码至少 6 位")
		return
	}
	adminID := c.GetInt64("admin_id")
	var hash string
	s.db.QueryRow("SELECT password_hash FROM admins WHERE id = ?", adminID).Scan(&hash)
	if !auth.CheckPassword(hash, req.OldPassword) {
		adminErr(c, http.StatusBadRequest, "原密码错误")
		return
	}
	s.db.Exec("UPDATE admins SET password_hash = ? WHERE id = ?", auth.HashPassword(req.NewPassword), adminID)
	s.logOp(adminID, c.GetString("admin_name"), "change_password", "", "")
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

// appKey 返回 App 接入所需的 app_id / app_secret（管理端登录后可见）
func (s *Server) appKey(c *gin.Context) {
	adminOK(c, gin.H{
		"app_id":       s.appID(),
		"app_secret":  s.appSecret(),
		"grace_hours":  s.graceHours(),
	})
}

func (s *Server) logOp(adminID int64, adminName, action, target, detail string) {
	s.db.Exec("INSERT INTO op_logs(admin_id, admin_name, action, target, detail, created_at) VALUES(?,?,?,?,?,?)",
		adminID, adminName, action, target, detail, time.Now().Unix())
}
