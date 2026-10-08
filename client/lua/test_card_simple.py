# -*- coding: utf-8 -*-
"""card.lua（简版）快速验证：mock 流程 + 线上真实负路径。"""
import sys

from lupa import LuaRuntime

LUA_FILE = r"d:\traeproject\card-system\client\lua\card.lua"

passed, failed = 0, 0


def check(name, actual, expected):
    global passed, failed
    if actual == expected:
        passed += 1
        print("PASS  %s" % name)
    else:
        failed += 1
        print("FAIL  %s\n  expect: %r\n  actual: %r" % (name, expected, actual))


with open(LUA_FILE, "r", encoding="utf-8") as f:
    src = f.read()

lua = LuaRuntime(unpack_returned_tuples=True)
g = lua.globals()

# ---- 1. 加载文件（全局函数落地）----
lua.execute(src)
check("load[verify_card 存在]", g.verify_card is not None, True)
check("load[heartbeat 存在]", g.heartbeat is not None, True)

# ---- 2. mock 流程 ----
mock = r'''
local responses = {
    '{"code":0,"msg":"ok","data":{"plan":"月卡","days_left":30,"is_test":false}}',
    '{"code":1002,"msg":"expired"}',
    '{"code":0,"msg":"ok","data":{"plan":"测试卡 · 24h","days_left":0,"is_test":true}}',
}
local i = 0
set_http(function(method, url, headers, body)
    i = i + 1
    return 200, responses[i]
end)
local ok1, msg1, days1 = verify_card("TEST-AAAA-BBBB-CCCC")
local ok2, msg2, days2, code2 = heartbeat("TEST-AAAA-BBBB-CCCC")
local ok3, msg3, days3 = heartbeat("TEST-DDDD-EEEE-FFFF")
return {ok1, msg1, days1, ok2, code2, msg2, ok3, days3}
'''
res = lua.execute(mock)
vals = [res[i] for i in range(1, len(res) + 1)]
check("mock[verify 成功]", bool(vals[0]) and vals[2] == 30, True)
check("mock[verify 成功消息]", vals[1], "激活成功，剩余 30 天")
check("mock[heartbeat 过期]", vals[3] == False and vals[4] == 1002, True)
check("mock[heartbeat 过期消息]", vals[5], "卡密已过期")
check("mock[恢复后成功]", bool(vals[6]), True)

# ---- 3. 线上真实负路径（注入 Python HTTP 桥，签名正确应返回 1001 而非 1007）----
import urllib.error
import urllib.request


def py_http(method, url, headers, body):
    h = {str(k): str(headers[k]) for k in headers}
    req = urllib.request.Request(url, data=body.encode("utf-8") if body else None, method=method)
    for k, v in h.items():
        if k.lower() != "content-length":
            req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, r.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace")


g.py_http_bridge = py_http
lua.execute("set_http(py_http_bridge)")
res = lua.execute('''
local ok, msg, days, code = verify_card("TEST-0000-0000-0000")
return {ok, msg, days, code}
''')
vals = [res[i] for i in range(1, len(res) + 1)]
check("live[无效卡 -> 1001（签名被接受）]", vals[0] == False and vals[3] == 1001, True)
check("live[错误消息]", vals[1], "无效卡密")

print("\n%d passed, %d failed" % (passed, failed))
sys.exit(1 if failed else 0)
