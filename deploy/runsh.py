import sys, json, paramiko

from secrets import HOST, PORT, USER, PASSWORD  # 真实值在 secrets.py（不入库，模板见 secrets.py.example）

def main(script_path, timeout=900):
    with open(script_path, "r", encoding="utf-8") as f:
        script = f.read()
    cli = paramiko.SSHClient()
    cli.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    cli.connect(HOST, port=PORT, username=USER, password=PASSWORD, timeout=60, banner_timeout=60, auth_timeout=60)
    try:
        sftp = cli.open_sftp()
        with sftp.open("/root/deploy_run.sh", "w") as rf:
            rf.write(script)
        sftp.close()
        stdin, stdout, stderr = cli.exec_command("bash /root/deploy_run.sh", timeout=timeout)
        rc = stdout.channel.recv_exit_status()
        out = stdout.read().decode("utf-8", "replace")
        err = stderr.read().decode("utf-8", "replace")
        print(json.dumps({"rc": rc, "out": out, "err": err[-2000:]}, ensure_ascii=False))
    finally:
        cli.close()

if __name__ == "__main__":
    main(sys.argv[1])
