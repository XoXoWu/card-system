package api

import (
	"net/http"
	"time"

	"card-system/internal/model"

	"github.com/gin-gonic/gin"
)

type planReq struct {
	Name       string  `json:"name" binding:"required"`
	Days       int     `json:"days" binding:"required,min=1,max=3650"`
	MaxDevices int     `json:"max_devices" binding:"required,min=1,max=10"`
	Price      float64 `json:"price" binding:"min=0"`
	Remark     string  `json:"remark"`
	Enabled    *bool   `json:"enabled"`
}

func (s *Server) listPlans(c *gin.Context) {
	rows, err := s.db.Query(`SELECT id, name, days, max_devices, price, remark, enabled, created_at, updated_at FROM plans ORDER BY days`)
	if err != nil {
		adminErr(c, http.StatusInternalServerError, "查询套餐失败")
		return
	}
	defer rows.Close()
	list := []*model.Plan{}
	for rows.Next() {
		var p model.Plan
		var enabled int
		if err := rows.Scan(&p.ID, &p.Name, &p.Days, &p.MaxDevices, &p.Price, &p.Remark, &enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		p.Enabled = enabled == 1
		list = append(list, &p)
	}
	adminOK(c, list)
}

func (s *Server) createPlan(c *gin.Context) {
	var req planReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminErr(c, http.StatusBadRequest, "套餐参数不合法")
		return
	}
	var exists int
	s.db.QueryRow("SELECT 1 FROM plans WHERE name = ?", req.Name).Scan(&exists)
	if exists == 1 {
		adminErr(c, http.StatusBadRequest, "套餐名称已存在")
		return
	}
	now := time.Now().Unix()
	enabled := 1
	if req.Enabled != nil && !*req.Enabled {
		enabled = 0
	}
	res, err := s.db.Exec("INSERT INTO plans(name, days, max_devices, price, remark, enabled, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)",
		req.Name, req.Days, req.MaxDevices, req.Price, req.Remark, enabled, now, now)
	if err != nil {
		adminErr(c, http.StatusInternalServerError, "创建套餐失败")
		return
	}
	id, _ := res.LastInsertId()
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "plan_create", req.Name, "")
	adminOK(c, gin.H{"id": id})
}

func (s *Server) updatePlan(c *gin.Context) {
	var req planReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminErr(c, http.StatusBadRequest, "套餐参数不合法")
		return
	}
	var exists int
	s.db.QueryRow("SELECT 1 FROM plans WHERE name = ? AND id != ?", req.Name, c.Param("id")).Scan(&exists)
	if exists == 1 {
		adminErr(c, http.StatusBadRequest, "套餐名称已存在")
		return
	}
	enabled := 1
	if req.Enabled != nil && !*req.Enabled {
		enabled = 0
	}
	_, err := s.db.Exec("UPDATE plans SET name=?, days=?, max_devices=?, price=?, remark=?, enabled=?, updated_at=? WHERE id=?",
		req.Name, req.Days, req.MaxDevices, req.Price, req.Remark, enabled, time.Now().Unix(), c.Param("id"))
	if err != nil {
		adminErr(c, http.StatusInternalServerError, "更新套餐失败")
		return
	}
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "plan_update", req.Name, "")
	adminOK(c, "ok")
}

func (s *Server) deletePlan(c *gin.Context) {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM cards WHERE plan_id = ?", c.Param("id")).Scan(&count)
	if count > 0 {
		adminErr(c, http.StatusBadRequest, "该套餐下已有卡密，不可删除，可改为停用")
		return
	}
	res, _ := s.db.Exec("DELETE FROM plans WHERE id = ?", c.Param("id"))
	if n, _ := res.RowsAffected(); n == 0 {
		adminErr(c, http.StatusNotFound, "套餐不存在")
		return
	}
	s.logOp(c.GetInt64("admin_id"), c.GetString("admin_name"), "plan_delete", c.Param("id"), "")
	adminOK(c, "ok")
}
