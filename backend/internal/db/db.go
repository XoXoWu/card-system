package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS admins (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  username      TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS plans (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  name        TEXT NOT NULL UNIQUE,
  days        INTEGER NOT NULL,
  max_devices INTEGER NOT NULL DEFAULT 1,
  price       REAL NOT NULL DEFAULT 0,
  remark      TEXT NOT NULL DEFAULT '',
  enabled     INTEGER NOT NULL DEFAULT 1,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS cards (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  key_hash      TEXT NOT NULL UNIQUE,
  key_tail      TEXT NOT NULL,
  plan_id       INTEGER NOT NULL DEFAULT 0,
  plan_name     TEXT NOT NULL,
  is_test       INTEGER NOT NULL DEFAULT 0,
  duration_days INTEGER NOT NULL DEFAULT 30,
  max_devices   INTEGER NOT NULL DEFAULT 1,
  status        TEXT NOT NULL DEFAULT 'unused',
  batch_no      TEXT NOT NULL DEFAULT '',
  note          TEXT NOT NULL DEFAULT '',
  activated_at  INTEGER NOT NULL DEFAULT 0,
  expire_at     INTEGER NOT NULL DEFAULT 0,
  banned_at     INTEGER NOT NULL DEFAULT 0,
  revoked_at    INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_cards_status ON cards(status);
CREATE INDEX IF NOT EXISTS idx_cards_batch  ON cards(batch_no);
CREATE INDEX IF NOT EXISTS idx_cards_plan   ON cards(plan_id);

CREATE TABLE IF NOT EXISTS devices (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  card_id     INTEGER NOT NULL,
  device_id   TEXT NOT NULL,
  device_info TEXT NOT NULL DEFAULT '',
  first_seen  INTEGER NOT NULL DEFAULT 0,
  last_seen   INTEGER NOT NULL DEFAULT 0,
  unbound_at  INTEGER NOT NULL DEFAULT 0,
  UNIQUE(card_id, device_id)
);
CREATE INDEX IF NOT EXISTS idx_devices_card ON devices(card_id);

CREATE TABLE IF NOT EXISTS op_logs (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  admin_id   INTEGER NOT NULL DEFAULT 0,
  admin_name TEXT NOT NULL DEFAULT '',
  action     TEXT NOT NULL,
  target     TEXT NOT NULL DEFAULT '',
  detail     TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS nonces (
  nonce      TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

func Open(path string) *sql.DB {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(fmt.Sprintf("create data dir: %v", err))
		}
	}
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		panic(fmt.Sprintf("open sqlite: %v", err))
	}
	d.SetMaxOpenConns(1)
	if err := d.Ping(); err != nil {
		panic(fmt.Sprintf("ping sqlite: %v", err))
	}
	return d
}

func Migrate(d *sql.DB) {
	if _, err := d.Exec(schema); err != nil {
		panic(fmt.Sprintf("migrate: %v", err))
	}
}

func GetSetting(d *sql.DB, key string) string {
	var v string
	d.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	return v
}

func SetSetting(d *sql.DB, key, value string) {
	d.Exec("INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
}

func CleanExpiredNonces(d *sql.DB, before int64) {
	d.Exec("DELETE FROM nonces WHERE created_at < ?", before)
}
