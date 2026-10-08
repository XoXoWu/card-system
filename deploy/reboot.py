import time, paramiko, sys

from secrets import HOST, PORT, USER, PASSWORD  # 真实值在 secrets.py（不入库，模板见 secrets.py.example）

cli = paramiko.SSHClient()
cli.set_missing_host_key_policy(paramiko.AutoAddPolicy())
try:
    cli.connect(HOST, port=PORT, username=USER, password=PASSWORD, timeout=120, banner_timeout=120, auth_timeout=120)
    print("CONNECTED, sending reboot...", flush=True)
    # fire and forget; reboot kills the connection immediately
    cli.exec_command("nohup reboot >/dev/null 2>&1 &", timeout=5)
    print("REBOOT_SENT", flush=True)
except Exception as e:
    print(f"FAILED: {type(e).__name__}: {e}")
    sys.exit(1)
finally:
    try:
        cli.close()
    except Exception:
        pass
