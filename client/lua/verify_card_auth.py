# -*- coding: utf-8 -*-
"""验证 card_auth.lua：SHA-256 / HMAC-SHA256 / JSON / 签名 / 心跳管理器。"""
import hashlib
import hmac as py_hmac
import json
import sys

from lupa import LuaRuntime

LUA_FILE = r"d:\traeproject\card-system\client\lua\card_auth.lua"

with open(LUA_FILE, "r", encoding="utf-8") as f:
    lua_code = f.read()

lua = LuaRuntime(unpack_returned_tuples=True)
card_auth = lua.execute("return (function()\n" + lua_code + "\nend)()")

passed, failed = 0, 0


def check(name, actual, expected):
    global passed, failed
    if actual == expected:
        passed += 1
        print("PASS  %s" % name)
    else:
        failed += 1
        print("FAIL  %s\n  expect: %r\n  actual: %r" % (name, expected, actual))


# ---------- SHA-256 标准测试向量 ----------
sha_vectors = [
    ("", "empty"),
    ("abc", "abc"),
    ("abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq", "448-bit"),
    ("The quick brown fox jumps over the lazy dog", "fox"),
    ("a" * 200, "multi-block"),
    ("中文测试消息，包含特殊字符 \"quote\" \\backslash\n newline", "utf8+ctrl"),
    ("x" * 55, "pad-boundary-55"),
    ("x" * 56, "pad-boundary-56"),
    ("x" * 64, "pad-boundary-64"),
    ("x" * 119, "pad-boundary-119"),
]
for msg, label in sha_vectors:
    expected = hashlib.sha256(msg.encode("utf-8")).hexdigest()
    check("sha256[%s]" % label, card_auth.sha256_hex(msg), expected)

# ---------- HMAC-SHA256（RFC 4231，二进制用 bytes 传给 Lua，1:1 字节） ----------
hmac_vectors = [
    (b"\x0b" * 20, b"Hi There", "RFC4231-TC1"),
    (b"Jefe", b"what do ya want for nothing?", "RFC4231-TC2"),
    (b"\xaa" * 20, b"\xdd" * 50, "RFC4231-TC3"),
    (bytes(range(0x01, 0x1A)), b"\xcd" * 50, "RFC4231-TC4"),
    (b"\xaa" * 131, b"Test Using Larger Than Block-Size Key - Hash Key First", "RFC4231-TC6"),
    (b"dev-secret-change-me", "1700000000000n-1-2-3{\"card_key\":\"TEST-ABCD-EFGH-IJKL\"}".encode(), "signed-req-sample"),
]
for key, msg, label in hmac_vectors:
    expected = py_hmac.new(key, msg, hashlib.sha256).hexdigest()
    check("hmac[%s]" % label, card_auth.hmac_sha256_hex(key, msg), expected)

# ---------- 签名一致性：Lua 生成的请求头签名 vs Python 独立计算 ----------
req = card_auth.build_request("activate", lua.table_from({
    "card_key": "TEST-ABCD-EFGH-IJKL",
    "device_id": "dev-1",
    "device_info": "中文设备信息 Windows 11",
}))
body = req["body"]
ts = req["headers"]["X-Timestamp"]
nonce = req["headers"]["X-Nonce"]
check("format[X-Timestamp 为纯数字毫秒]", bool(ts), True)
check("format[X-Timestamp 非科学计数法]", "e+" not in ts and "e-" not in ts, True)
expected_sig = py_hmac.new(b"dev-secret-change-me", (ts + nonce + body).encode("utf-8"), hashlib.sha256).hexdigest()
check("signature[build_request]", req["headers"]["X-Signature"], expected_sig)

# ---------- JSON 编码 ----------
parsed = json.loads(body)
check("json.encode[activate-body]",
      json.dumps(parsed, sort_keys=True, ensure_ascii=False),
      json.dumps({"card_key": "TEST-ABCD-EFGH-IJKL", "device_id": "dev-1", "device_info": "中文设备信息 Windows 11"},
                 sort_keys=True, ensure_ascii=False))

# ---------- JSON 解析（中文 + 特殊字符往返） ----------
resp_json = json.dumps({"code": 0, "msg": "成功",
                        "data": {"plan": "月卡", "is_test": True, "days_left": 30,
                                 "note": "引号\"与\\反斜杠\n换行",
                                 "devices": {"bound": 1, "max": 3}}},
                       ensure_ascii=False)
r = card_auth.parse_response(resp_json)
ok = (r["ok"] == True and r["code"] == 0 and r["msg"] == "成功"
      and r["data"]["plan"] == "月卡" and r["data"]["note"] == '引号"与\\反斜杠\n换行'
      and r["data"]["devices"]["max"] == 3)
check("json.decode[chinese-roundtrip]", ok, True)

bad = card_auth.parse_response("not-json{{{")
check("json.decode[bad-input-tolerant]", bad["ok"] == False and bad["code"] == -1, True)

empty = card_auth.parse_response(None)
check("json.decode[empty-tolerant]", empty["ok"] == False, True)

# ---------- 错误码 ----------
check("errcode[is_fatal(1001)]", card_auth.is_fatal(1001), True)
check("errcode[is_fatal(1002)]", card_auth.is_fatal(1002), True)
check("errcode[is_fatal(0)]", card_auth.is_fatal(0), False)
check("errcode[is_fatal(1020)]", card_auth.is_fatal(1020), False)

# ---------- 心跳管理器：mock HTTP 全流程（成功×2 → 过期 → 停止后不再发） ----------
heartbeat_lua = r"""
local results = {}
local fake_now = 1000000
os.time = function() return fake_now end

local responses = {
    '{"code":0,"msg":"ok","data":{"days_left":30}}',
    '{"code":0,"msg":"ok","data":{"days_left":29}}',
    '{"code":1002,"msg":"expired"}',
}
local ridx = 0
card_auth.set_http_func(function(method, url, headers, body)
    ridx = ridx + 1
    results[#results+1] = { what = "http", url = url }
    return 200, responses[ridx]
end)

card_auth.init({
    base_url = "http://mock-server",
    app_id = "my-app",
    app_secret = "sec",
    device_id = "dev-1",
    device_info = "mock",
})

card_auth.heartbeat_start("TEST-AAAA-BBBB-CCCC", 300, function(r, fails)
    results[#results+1] = { what = "cb", code = r.code, fails = fails }
end)

card_auth.heartbeat_tick()                    -- 第 1 次：成功
fake_now = fake_now + 301
card_auth.heartbeat_tick()                    -- 第 2 次：成功
fake_now = fake_now + 301
card_auth.heartbeat_tick()                    -- 第 3 次：过期
card_auth.heartbeat_stop()
fake_now = fake_now + 301
card_auth.heartbeat_tick()                    -- 已停止：不应发请求

local info = card_auth.heartbeat_info()
results[#results+1] = { what = "final",
    running = info.running, fail_count = info.fail_count,
    last_code = info.last_result and info.last_result.code or -999 }
results[#results+1] = { what = "count_snapshot", http = ridx }

-- 响应缺失（nil body）不应 panic
card_auth.heartbeat_start("TEST-AAAA-BBBB-CCCC", 300)
fake_now = fake_now + 301
card_auth.heartbeat_tick()

return results
"""
lua.globals().card_auth = card_auth
results = lua.execute(heartbeat_lua)
res_list = [results[i] for i in range(1, len(results) + 1)]

http_calls = [x for x in res_list if x["what"] == "http"]
callbacks = [x for x in res_list if x["what"] == "cb"]
final = [x for x in res_list if x["what"] == "final"][0]
snapshot = [x for x in res_list if x["what"] == "count_snapshot"][0]["http"]

check("heartbeat[url 正确]", http_calls[0]["url"], "http://mock-server/api/v1/verify")
check("heartbeat[cb1 成功且 fail=0]", callbacks[0]["code"] == 0 and callbacks[0]["fails"] == 0, True)
check("heartbeat[cb2 成功且 fail=0]", callbacks[1]["code"] == 0 and callbacks[1]["fails"] == 0, True)
check("heartbeat[cb3 过期且 fail=1]", callbacks[2]["code"] == 1002 and callbacks[2]["fails"] == 1, True)
check("heartbeat[stop 后不发请求]", snapshot, 3)
check("heartbeat[nil 响应不 panic]", len(http_calls), 4)
check("heartbeat[info 最后状态]", final["running"] == False and final["fail_count"] == 1 and final["last_code"] == 1002, True)

print("\n%d passed, %d failed" % (passed, failed))
sys.exit(1 if failed else 0)
