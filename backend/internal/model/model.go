package model

type Plan struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Days       int     `json:"days"`
	MaxDevices int     `json:"max_devices"`
	Price      float64 `json:"price"`
	Remark     string  `json:"remark"`
	Enabled    bool    `json:"enabled"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
}

type Card struct {
	ID           int64  `json:"id"`
	KeyTail      string `json:"key_tail"`
	PlanID       int64  `json:"plan_id"`
	PlanName     string `json:"plan_name"`
	IsTest       bool   `json:"is_test"`
	DurationDays int    `json:"duration_days"`
	MaxDevices   int    `json:"max_devices"`
	Status       string `json:"status"` // unused / active / banned / revoked / expired(动态)
	BatchNo      string `json:"batch_no"`
	Note         string `json:"note"`
	ActivatedAt  int64  `json:"activated_at"`
	ExpireAt     int64  `json:"expire_at"`
	BannedAt     int64  `json:"banned_at"`
	RevokedAt    int64  `json:"revoked_at"`
	CreatedAt    int64  `json:"created_at"`
	DeviceCount  int    `json:"device_count"`
}

type Device struct {
	ID         int64  `json:"id"`
	CardID     int64  `json:"card_id"`
	DeviceID   string `json:"device_id"`
	DeviceInfo string `json:"device_info"`
	FirstSeen  int64  `json:"first_seen"`
	LastSeen   int64  `json:"last_seen"`
	UnboundAt  int64  `json:"unbound_at"`
}

type Admin struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	CreatedAt    int64  `json:"created_at"`
}

type OpLog struct {
	ID        int64  `json:"id"`
	AdminID   int64  `json:"admin_id"`
	AdminName string `json:"admin_name"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	Detail    string `json:"detail"`
	CreatedAt int64  `json:"created_at"`
}

// App 端响应码，见 PRD 7.5
const (
	CodeOK          = 0
	CodeBadRequest  = 1009
	CodeSignError   = 1007
	CodeReplay      = 1008
	CodeTooMany     = 1020
	CodeInvalidCard = 1001
	CodeExpired     = 1002
	CodeBanned      = 1003
	CodeRevoked     = 1004
	CodeDeviceLimit = 1005
	CodeUnbound     = 1006
	CodeCardUnused  = 1011 // verify/renew 时卡未激活
)

func MsgOf(code int) string {
	switch code {
	case CodeOK:
		return "ok"
	case CodeBadRequest:
		return "bad_request"
	case CodeSignError:
		return "sign_error"
	case CodeReplay:
		return "replay_detected"
	case CodeTooMany:
		return "too_many_requests"
	case CodeInvalidCard:
		return "invalid_card"
	case CodeExpired:
		return "expired"
	case CodeBanned:
		return "banned"
	case CodeRevoked:
		return "revoked"
	case CodeDeviceLimit:
		return "device_limit"
	case CodeUnbound:
		return "device_unbound"
	case CodeCardUnused:
		return "card_unused"
	default:
		return "error"
	}
}
