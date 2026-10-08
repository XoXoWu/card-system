import json, paramiko

from secrets import HOST, PORT, USER, PASSWORD  # 真实值在 secrets.py（不入库，模板见 secrets.py.example）
LOCAL_ZIP = r"d:\traeproject\card-system\deploy\card-system.zip"
REMOTE_ZIP = "/root/card-system.zip"

cli = paramiko.SSHClient()
cli.set_missing_host_key_policy(paramiko.AutoAddPolicy())
cli.connect(HOST, port=PORT, username=USER, password=PASSWORD, timeout=25, banner_timeout=25, auth_timeout=25)

sftp = cli.open_sftp()
print("uploading", LOCAL_ZIP, "->", REMOTE_ZIP)
sftp.put(LOCAL_ZIP, REMOTE_ZIP)
st = sftp.stat(REMOTE_ZIP)
print("uploaded bytes:", st.st_size)
sftp.close()

stdin, stdout, stderr = cli.exec_command("ls -la /root/card-system.zip && md5sum /root/card-system.zip", timeout=60)
print(stdout.read().decode("utf-8", "replace"))
print(stderr.read().decode("utf-8", "replace"))
cli.close()
print("OK")
