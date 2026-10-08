import time, json, paramiko, sys

from secrets import HOST, PORT, USER, PASSWORD  # 真实值在 secrets.py（不入库，模板见 secrets.py.example）

def try_connect(attempts=1, wait=10):
    for i in range(attempts):
        cli = paramiko.SSHClient()
        cli.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        try:
            cli.connect(HOST, port=PORT, username=USER, password=PASSWORD, timeout=90, banner_timeout=90, auth_timeout=90)
            return cli
        except Exception as e:
            print(f"attempt {i+1}/{attempts}: {type(e).__name__}: {e}", flush=True)
            try:
                cli.close()
            except Exception:
                pass
            if i < attempts - 1:
                time.sleep(wait)
    return None

if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "uptime && free -h"
    cli = try_connect()
    if cli is None:
        print("CONNECT_FAILED")
        sys.exit(1)
    try:
        stdin, stdout, stderr = cli.exec_command(cmd, timeout=120)
        rc = stdout.channel.recv_exit_status()
        print(json.dumps({"rc": rc, "out": stdout.read().decode("utf-8", "replace"), "err": stderr.read().decode("utf-8", "replace")}, ensure_ascii=False))
    finally:
        cli.close()
