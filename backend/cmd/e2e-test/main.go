package main

// App 验证 API 端到端测试：activate / verify / renew / 签名与防重放

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	baseURL  = "http://127.0.0.1:18080/api/v1"
	appID    = "my-app"
	secret   = "change-me-in-production-app-secret"
	cardKey  = "CARD_KEY"
	testKey  = "TEST_KEY"
	cardKey2 = "CARD_KEY_2"
)

var failed int

func check(name string, ok bool, extra ...any) {
	if ok {
		fmt.Printf("PASS  %s %v\n", name, extra)
	} else {
		failed++
		fmt.Printf("FAIL  %s %v\n", name, extra)
	}
}

func sign(body string) (ts, nonce, sig string) {
	ts = fmt.Sprintf("%d", time.Now().UnixMilli())
	nonce = fmt.Sprintf("n%d%d", time.Now().UnixNano(), rand.Intn(1e6))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + nonce + body))
	return ts, nonce, hex.EncodeToString(mac.Sum(nil))
}

func call(endpoint, body string) map[string]any {
	return callWith(endpoint, body, nil)
}

func callWith(endpoint, body string, override map[string]string) map[string]any {
	ts, nonce, sig := sign(body)
	if v, ok := override["ts"]; ok {
		ts = v
	}
	if v, ok := override["nonce"]; ok {
		nonce = v
	}
	if v, ok := override["sig"]; ok {
		sig = v
	}
	req, _ := http.NewRequest("POST", baseURL+endpoint, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-App-Id", appID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return map[string]any{"code": -1, "msg": err.Error()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}

func postJSON(url, body string) (int, []byte) {
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func main() {
	normalKey := os.Getenv("NORMAL_KEY")
	testCard := os.Getenv("TEST_KEY")
	renewKey := os.Getenv("RENEW_KEY")

	// 1. 激活普通卡
	r := call("/activate", fmt.Sprintf(`{"card_key":%q,"device_id":"dev-001","device_info":"Windows 11 · x64"}`, normalKey))
	check("activate 普通卡", r["code"].(float64) == 0, r)
	days := r["data"].(map[string]any)["days_left"].(float64)
	check("月卡剩余约 30 天", days >= 29 && days <= 30, days)

	// 2. 重复激活同一设备 → 幂等成功
	r = call("/activate", fmt.Sprintf(`{"card_key":%q,"device_id":"dev-001","device_info":"Windows 11"}`, normalKey))
	check("activate 幂等", r["code"].(float64) == 0, r["code"])

	// 3. 第二台设备激活（月卡 max=1）→ device_limit
	r = call("/activate", fmt.Sprintf(`{"card_key":%q,"device_id":"dev-002","device_info":"Android 14"}`, normalKey))
	check("月卡第二台设备拒绝", r["code"].(float64) == 1005, r["code"])

	// 4. 心跳验证
	r = call("/verify", fmt.Sprintf(`{"card_key":%q,"device_id":"dev-001"}`, normalKey))
	check("verify 心跳", r["code"].(float64) == 0, r["data"])

	// 5. 无效卡
	r = call("/activate", `{"card_key":"AAAA-BBBB-CCCC-DDDD","device_id":"dev-x","device_info":"t"}`)
	check("无效卡 invalid_card", r["code"].(float64) == 1001, r["code"])

	// 6. 签名错误
	r = callWith("/verify", fmt.Sprintf(`{"card_key":%q,"device_id":"dev-001"}`, normalKey), map[string]string{"sig": "bad"})
	check("签名错误拒绝", r["code"].(float64) == 1007, r["code"])

	// 7. 防重放：同 nonce 两次请求
	body := fmt.Sprintf(`{"card_key":%q,"device_id":"dev-001"}`, normalKey)
	t2, n2, s2 := sign(body)
	r1 := callWith("/verify", body, map[string]string{"ts": t2, "nonce": n2, "sig": s2})
	r2 := callWith("/verify", body, map[string]string{"ts": t2, "nonce": n2, "sig": s2})
	check("防重放第二次拒绝", r1["code"].(float64) == 0 && r2["code"].(float64) == 1008, r2["code"])

	// 8. 时间戳偏差 > 5 分钟
	oldTs := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).UnixMilli())
	b3 := fmt.Sprintf(`{"card_key":%q,"device_id":"dev-001"}`, normalKey)
	_, nn, _ := sign(b3)
	r = callWith("/verify", b3, map[string]string{"ts": oldTs, "nonce": nn, "sig": "x"})
	check("时间戳过期拒绝(先于签名校验)", r["code"].(float64) == 1007, r["code"])

	// 9. 测试卡：激活后 24h 生效、限 1 台
	r = call("/activate", fmt.Sprintf(`{"card_key":%q,"device_id":"dev-t1","device_info":"内测设备"}`, testCard))
	check("测试卡激活", r["code"].(float64) == 0, r["data"])
	tdata := r["data"].(map[string]any)
	check("测试卡 24 小时有效", tdata["days_left"].(float64) == 1, tdata["days_left"])
	r = call("/activate", fmt.Sprintf(`{"card_key":%q,"device_id":"dev-t2","device_info":"内测设备2"}`, testCard))
	check("测试卡限 1 台", r["code"].(float64) == 1005, r["code"])

	// 10. 测试卡不可作为续费卡
	r = call("/renew", fmt.Sprintf(`{"current_card_key":%q,"new_card_key":%q,"device_id":"dev-001"}`, normalKey, testCard))
	check("测试卡不可续费", r["code"].(float64) == 1004, r["code"])

	// 11. 正常续费：新卡时长叠加
	r = call("/renew", fmt.Sprintf(`{"current_card_key":%q,"new_card_key":%q,"device_id":"dev-001"}`, normalKey, renewKey))
	check("续费成功", r["code"].(float64) == 0, r["data"])
	rd := r["data"].(map[string]any)
	check("叠加后约 60 天", rd["days_left"].(float64) >= 59 && rd["days_left"].(float64) <= 60, rd["days_left"])

	// 12. 已消耗的续费卡再次使用 → invalid
	r = call("/renew", fmt.Sprintf(`{"current_card_key":%q,"new_card_key":%q,"device_id":"dev-001"}`, normalKey, renewKey))
	check("续费卡不可重复使用", r["code"].(float64) == 1004, r["code"])

	fmt.Printf("\n===== %s (失败 %d 项) =====\n", map[bool]string{true: "全部通过", false: "存在失败"}[failed == 0], failed)
	if failed > 0 {
		os.Exit(1)
	}
	_ = bytes.MinRead
}
