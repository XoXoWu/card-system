# -*- coding: utf-8 -*-
"""在所有可用的 lupa Lua 运行时上跑核心向量测试（5.1/5.2/5.3/5.4/5.5/LuaJIT）。"""
import hashlib
import hmac as py_hmac
import json
import traceback

import lupa

LUA_FILE = r"d:\traeproject\card-system\client\lua\card_auth.lua"
RUNTIMES = ["lua51", "lua52", "lua53", "lua54", "lua55"]

with open(LUA_FILE, "r", encoding="utf-8") as f:
    src = f.read()


def load(rt_name):
    rt_mod = getattr(lupa, rt_name)
    lua = rt_mod.LuaRuntime(encoding="utf-8")
    mod = lua.execute("return (function()\n" + src + "\nend)()")
    return lua, mod


def tdict(t):
    return {str(k): t[k] for k in t.keys()}


sha_cases = ["", "abc", "a" * 200, "中文测试", "x" * 55, "x" * 64, "x" * 119]
hmac_cases = [(b"key", b"The quick brown fox jumps over the lazy dog"),
              (b"k" * 100, b"long key case"),
              (b"sec", "中文消息含\"引号\"".encode("utf-8"))]

for rt in RUNTIMES:
    try:
        lua, mod = load(rt)
    except Exception as e:
        print("[%s] 加载失败: %s" % (rt, e))
        traceback.print_exc()
        continue
    ok = True
    try:
        for msg in sha_cases:
            if mod.sha256_hex(msg) != hashlib.sha256(msg.encode("utf-8")).hexdigest():
                ok = False
                print("  [%s] sha256 FAIL: %r" % (rt, msg[:20]))
        for key, msg in hmac_cases:
            if mod.hmac_sha256_hex(key, msg) != py_hmac.new(key, msg, hashlib.sha256).hexdigest():
                ok = False
                print("  [%s] hmac FAIL key=%r" % (rt, key[:12]))
        # 请求构造：时间戳格式 + 签名一致性 + JSON 中文
        req = mod.build_request("activate", lua.table_from({
            "card_key": "TEST-ABCD-EFGH-IJKL",
            "device_id": "dev-1",
            "device_info": "中文设备信息",
        }))
        body, ts, nonce = req["body"], req["headers"]["X-Timestamp"], req["headers"]["X-Nonce"]
        sig_expect = py_hmac.new(b"dev-secret-change-me", (ts + nonce + body).encode("utf-8"), hashlib.sha256).hexdigest()
        if req["headers"]["X-Signature"] != sig_expect:
            ok = False
            print("  [%s] signature FAIL" % rt)
        if not ts.isdigit():
            ok = False
            print("  [%s] ts 非纯数字: %r" % (rt, ts))
        parsed = json.loads(body)
        if parsed["device_info"] != "中文设备信息":
            ok = False
            print("  [%s] json encode FAIL: %s" % (rt, body))
        # JSON 解码中文
        sample = json.dumps({"code": 0, "msg": "成功", "data": {"plan": "月卡", "devices": {"bound": 1, "max": 3}}}, ensure_ascii=False)
        r = mod.parse_response(sample)
        d = tdict(r["data"])
        if r["code"] != 0 or tdict(d["devices"])["max"] != 3:
            ok = False
            print("  [%s] json decode FAIL" % rt)
    except Exception as e:
        ok = False
        print("  [%s] 异常: %s" % (rt, e))
        traceback.print_exc()
    print("[%-7s] %s" % (rt, "PASS" if ok else "FAIL"))
