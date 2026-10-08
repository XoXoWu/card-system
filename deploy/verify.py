import os, urllib.request, json

BASE = os.environ.get("CARD_BASE", "http://127.0.0.1:8080")

def req(method, path, body=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    resp = urllib.request.urlopen(r, timeout=20)
    return resp.status, resp.read()

s, b = req("POST", "/api/admin/login", {"username": os.environ.get("CARD_ADMIN_USER", "admin"),
                                        "password": os.environ.get("CARD_ADMIN_PASSWORD", "admin123")})
tok = json.loads(b)["data"]["token"]
print("1. login:", s, "token:", tok[:25] + "...")

s, b = req("GET", "/api/admin/plans", token=tok)
data = json.loads(b)["data"]
plans = data["items"] if isinstance(data, dict) else data
print("2. plans:", s, [p["name"] for p in plans])

s, b = req("GET", "/api/admin/stats/overview", token=tok)
print("3. stats:", s, b.decode()[:120])

s, b = req("POST", "/api/admin/cards/test", {"count": 1, "note": "deploy check"}, token=tok)
print("4. test card:", s, b.decode()[:200])
