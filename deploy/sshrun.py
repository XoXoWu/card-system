import sys, json, paramiko

from secrets import HOST, PORT, USER, PASSWORD  # 真实值在 secrets.py（不入库，模板见 secrets.py.example）

def get_client():
    cli = paramiko.SSHClient()
    cli.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    cli.connect(HOST, port=PORT, username=USER, password=PASSWORD, timeout=25, banner_timeout=25, auth_timeout=25)
    return cli

def run(cmd, timeout=600):
    cli = get_client()
    try:
        stdin, stdout, stderr = cli.exec_command(cmd, timeout=timeout)
        rc = stdout.channel.recv_exit_status()
        out = stdout.read().decode("utf-8", "replace")
        err = stderr.read().decode("utf-8", "replace")
        return {"rc": rc, "out": out, "err": err}
    finally:
        cli.close()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("usage: sshrun.py <command>")
        sys.exit(2)
    r = run(sys.argv[1])
    print(json.dumps(r, ensure_ascii=False))
