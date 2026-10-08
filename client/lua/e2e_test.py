# -*- coding: utf-8 -*-
"""线上端到端验证：card_auth.lua 对接真实服务器。
用法：
  python e2e_test.py                 # 负路径测试（不消耗卡密）
  python e2e_test.py <卡密>          # 完整流程：激活 + 心跳 + 设备上限
"""
import os
import sys
import time
import urllib.error
import urllib.request

from lupa import LuaRuntime

# 目标与凭据从环境变量读取，不写入仓库：
#   CARD_BASE（默认本机 8080）、CARD_APP_ID、CARD_APP_SECRET（必填才能过签名）
BASE = os.environ.get("CARD_BASE", "http://127.0.0.1:8080")
APP_ID = os.environ.get("CARD_APP_ID", "my-app")
APP_SECRET = os.environ.get("CARD_APP_SECRET", "")
LUA_FILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "card_auth.lua")

passed, failed = 0, 0


def check(name, actual, expected):
    global passed, failed
    if actual == expected:
        passed += 1
        print("PASS  %s" % name)
    else:
        failed += 1
        print("FAIL  %s\n  expect: %r\n  actual: %r" % (name, expected, actual))


lua = LuaRuntime(unpack_returned_tuples=True)
with open(LUA_FILE, "r", encoding="utf-8") as f:
    lua_code = f.read()
card_auth = lua.execute("return (function()\n" + lua_code + "\nend)()")


def lua_http(method, url, headers, body):
    h = {}
    for k in headers:
        h[str(k)] = str(headers[k])
    req = urllib.request.Request(url, data=body.encode("utf-8") if body else None, method=method)
    for k, v in h.items():
        if k.lower() != "content-length":
            req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, r.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace")


card_auth.set_http_func(lua_http)
card_auth.init(lua.table_from({
    "base_url": BASE,
    "app_id": APP_ID,
    "app_secret": APP_SECRET,
    "device_id": "e2e-lua-device-001",
    "device_info": "Lua SDK e2e 测试",
}))

# ---- 负路径 1：无效卡密。若返回 1001 而非 1007，说明签名被服务器接受 ----
r = card_auth.activate("TEST-0000-0000-0000")
check("live[无效卡 -> 1001（签名被接受）]", r["ok"] == False and r["code"] == 1001, True)

# ---- 负路径 2：篡改签名 -> 1007 ----
req = card_auth.build_request("verify", lua.table_from({
    "card_key": "TEST-0000-0000-0000", "device_id": "e2e-lua-device-001"}))
req["headers"]["X-Signature"] = "0" * 64
status, body = lua_http(req["method"], req["url"], req["headers"], req["body"])
parsed = card_auth.parse_response(body)
check("live[篡改签名 -> 1007]", parsed["code"] == 1007, True)

# ---- 负路径 3：过期时间戳 -> 1007 ----
req = card_auth.build_request("verify", lua.table_from({
    "card_key": "TEST-0000-0000-0000", "device_id": "e2e-lua-device-001"}))
req["headers"]["X-Timestamp"] = "1000000000000"
status, body = lua_http(req["method"], req["url"], req["headers"], req["body"])
parsed = card_auth.parse_response(body)
check("live[过期时间戳 -> 1007]", parsed["code"] == 1007, True)

# ---- 正路径（可选）：需要一张真实卡密 ----
if len(sys.argv) > 1:
    card_key = sys.argv[1]
    t0 = time.time()
    r = card_auth.activate(card_key)
    print("      activate: ok=%s code=%s" % (bool(r["ok"]), r["code"]))
    if r["ok"]:
        d = r["data"]
        check("live[激活成功]", True, True)
        check("live[is_test]", d["is_test"] == True, True)
        check("live[24h]", 86390 < d["expire_at"] - t0 <= 86410, True)

        r2 = card_auth.verify(card_key)
        check("live[心跳 verify]", bool(r2["ok"]) and r2["code"] == 0, True)

        results = []
        card_auth.heartbeat_start(card_key, 60,
                                  lambda r, f: results.append(r["code"]))
        card_auth.heartbeat_tick()
        check("live[heartbeat 管理器]", results == [0], True)
        card_auth.heartbeat_stop()

        card_auth.init(lua.table_from({"device_id": "e2e-lua-device-002"}))
        r3 = card_auth.activate(card_key)
        check("live[设备上限 -> 1005]", r3["ok"] == False and r3["code"] == 1005, True)
    else:
        check("live[激活成功]", "code=%s msg=%s" % (r["code"], r["msg"]), "ok")
else:
    print("\n（未提供卡密，跳过正路径测试——用真实卡密运行可测完整流程）")

print("\n%d passed, %d failed" % (passed, failed))
sys.exit(1 if failed else 0)
