package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"card-system/internal/api"
	"card-system/internal/auth"
	"card-system/internal/config"
	"card-system/internal/db"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if os.Getenv("CARD_ENV") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	database := db.Open(cfg.DBPath)
	defer database.Close()
	db.Migrate(database)
	seedData(database, cfg)

	srv := api.NewServer(cfg, database)
	log.Printf("card-system listening on %s", cfg.Listen)
	if err := srv.Router().Run(cfg.Listen); err != nil {
		log.Fatal(err)
	}
}

func seedData(d *sql.DB, cfg *config.Config) {
	var adminCount int
	d.QueryRow("SELECT COUNT(*) FROM admins").Scan(&adminCount)
	if adminCount == 0 {
		d.Exec("INSERT INTO admins(username, password_hash, created_at) VALUES(?,?,?)",
			cfg.AdminUser, auth.HashPassword(cfg.AdminPassword), time.Now().Unix())
		log.Printf("初始管理员已创建：%s（请登录后尽快修改密码）", cfg.AdminUser)
	}

	var planCount int
	d.QueryRow("SELECT COUNT(*) FROM plans").Scan(&planCount)
	if planCount == 0 {
		now := time.Now().Unix()
		d.Exec("INSERT INTO plans(name, days, max_devices, price, remark, enabled, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)",
			"月卡", 30, 1, 19.9, "30 天 · 单设备", 1, now, now)
		d.Exec("INSERT INTO plans(name, days, max_devices, price, remark, enabled, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)",
			"季卡", 90, 2, 49.9, "90 天 · 双设备", 1, now, now)
		d.Exec("INSERT INTO plans(name, days, max_devices, price, remark, enabled, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)",
			"年卡", 365, 3, 159.9, "365 天 · 三设备", 1, now, now)
		log.Println("已初始化默认套餐：月卡 / 季卡 / 年卡")
	}

	if db.GetSetting(d, "app_id") == "" {
		db.SetSetting(d, "app_id", "my-app")
	}
	if db.GetSetting(d, "app_secret") == "" {
		db.SetSetting(d, "app_secret", cfg.JWTSecret+"-app-secret")
	}
	if db.GetSetting(d, "grace_hours") == "" {
		db.SetSetting(d, "grace_hours", "72")
	}
}
