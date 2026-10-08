# -*- coding: utf-8 -*-
"""card.lua 在 Lua 5.1 / 5.2 下的兼容性检查。"""
from lupa import lua51, lua52

for rt in (lua51, lua52):
    lua = rt.LuaRuntime(encoding="utf-8")
    with open(r"d:\traeproject\card-system\client\lua\card.lua", encoding="utf-8") as f:
        lua.execute(f.read())
    g = lua.globals()
    lua.execute(r"""
    set_http(function()
        return 200, '{"code":0,"msg":"ok","data":{"plan":"月卡","days_left":30}}'
    end)
    """)
    ok, msg, days, code, data = g.verify_card("TEST-AAAA-BBBB-CCCC")
    ok2, msg2, days2, code2, _ = g.heartbeat("TEST-AAAA-BBBB-CCCC")
    print("%s: verify=%s/%s heartbeat=%s/%s msg=%s" %
          (rt.__name__.split(".")[-1], bool(ok), days, bool(ok2), days2, msg))
    assert bool(ok) and days == 30 and bool(ok2) and days2 == 30
print("OK")
