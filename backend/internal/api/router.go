package api

import (
	"database/sql"
	"net/http"
	"os"
	"path"
	"strings"

	"card-system/internal/config"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg     *config.Config
	db      *sql.DB
	limiter *ipLimiter
}

func NewServer(cfg *config.Config, db *sql.DB) *Server {
	return &Server{cfg: cfg, db: db, limiter: newIPLimiter()}
}

func (s *Server) appID() string {
	if v := getSetting(s.db, "app_id"); v != "" {
		return v
	}
	return "my-app"
}

func (s *Server) appSecret() string {
	if v := getSetting(s.db, "app_secret"); v != "" {
		return v
	}
	return "dev-secret-change-me"
}

func (s *Server) graceHours() int {
	if v := getSetting(s.db, "grace_hours"); v != "" {
		n := 0
		for _, r := range v {
			if r < '0' || r > '9' {
				return 72
			}
			n = n*10 + int(r-'0')
		}
		return n
	}
	return 72
}

func getSetting(db *sql.DB, key string) string {
	var v string
	db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	return v
}

func pageParams(c *gin.Context) (int, int) {
	p := atoi(c.DefaultQuery("page", "1"))
	sz := atoi(c.DefaultQuery("size", "20"))
	if p < 1 {
		p = 1
	}
	if sz < 1 || sz > 100 {
		sz = 20
	}
	return p, sz
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func (s *Server) Router() *gin.Engine {
	r := gin.Default()
	r.MaxMultipartMemory = 2 << 20

	// App 验证 API：签名 + 防重放 + 频控
	v1 := r.Group("/api/v1", s.rateLimit(), s.signVerify())
	{
		v1.POST("/activate", s.activate)
		v1.POST("/verify", s.verify)
		v1.POST("/renew", s.renew)
	}

	// 管理端
	admin := r.Group("/api/admin")
	{
		admin.POST("/login", s.login)
		authed := admin.Group("", s.adminAuth())
		{
			authed.POST("/password", s.changePassword)
			authed.GET("/appkey", s.appKey)

			authed.GET("/plans", s.listPlans)
			authed.POST("/plans", s.createPlan)
			authed.PUT("/plans/:id", s.updatePlan)
			authed.DELETE("/plans/:id", s.deletePlan)

			authed.POST("/cards/generate", s.generate)
			authed.POST("/cards/test", s.generateTest)
			authed.GET("/cards", s.listCards)
			authed.GET("/cards/:id", s.cardDetail)
			authed.POST("/cards/:id/ban", s.banCard)
			authed.POST("/cards/:id/unban", s.unbanCard)
			authed.POST("/cards/:id/revoke", s.revokeCard)
			authed.POST("/cards/:id/note", s.cardNote)
			authed.POST("/cards/:id/devices/:deviceId/unbind", s.unbindDevice)

			authed.GET("/stats/overview", s.statsOverview)
			authed.GET("/stats/trend", s.statsTrend)
			authed.GET("/stats/plan-dist", s.statsPlanDist)
			authed.GET("/oplogs", s.listOpLogs)
		}
	}

	// 管理后台静态资源（frontend/dist 构建产物）
	staticDir := envOr("CARD_STATIC_DIR", "static")
	if st, err := os.Stat(staticDir); err == nil && st.IsDir() {
		r.NoRoute(func(c *gin.Context) {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			fp := path.Join(staticDir, path.Clean("/"+p))
			if _, err := os.Stat(fp); err == nil && !strings.HasSuffix(fp, "index.html") || p == "/" {
				c.File(fp)
				return
			}
			c.File(path.Join(staticDir, "index.html"))
		})
	}
	return r
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
