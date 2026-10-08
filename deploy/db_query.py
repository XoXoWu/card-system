# -*- coding: utf-8 -*-
"""只读查询：操作日志 + 管理员账号状态（不显示任何密码数据）。"""
import sys

sys.path.insert(0, r"d:\traeproject\card-system\deploy")
from sshrun import run

PY = r'''
import sqlite3, json
db = sqlite3.connect("/tmp/snapshot.db")
db.row_factory = sqlite3.Row
out = {}
out["op_logs"] = [dict(r) for r in db.execute(
    "SELECT admin_name,action,target,created_at FROM op_logs ORDER BY id DESC LIMIT 20")]
out["admins"] = [dict(r) for r in db.execute(
    "SELECT id,username,created_at FROM admins")]
print(json.dumps(out, ensure_ascii=False))
'''
cmd = (
    "docker cp card-app:/app/data/data.db /tmp/snapshot.db && "
    "docker cp card-app:/app/data/data.db-wal /tmp/snapshot.db-wal 2>/dev/null; "
    "docker cp card-app:/app/data/data.db-shm /tmp/snapshot.db-shm 2>/dev/null; "
    "python3 - <<'PYEOF'\n" + PY + "\nPYEOF\n"
    "rm -f /tmp/snapshot.db /tmp/snapshot.db-wal /tmp/snapshot.db-shm"
)
r = run(cmd)
print(r["out"])
if r["err"]:
    print("ERR:", r["err"][:800])
