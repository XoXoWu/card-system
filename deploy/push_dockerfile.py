import paramiko

from secrets import HOST, PORT, USER, PASSWORD  # 真实值在 secrets.py（不入库，模板见 secrets.py.example）

cli = paramiko.SSHClient()
cli.set_missing_host_key_policy(paramiko.AutoAddPolicy())
cli.connect(HOST, port=PORT, username=USER, password=PASSWORD, timeout=25, banner_timeout=25, auth_timeout=25)
sftp = cli.open_sftp()
sftp.put(r"d:\traeproject\card-system\backend\Dockerfile", "/opt/card-system/backend/Dockerfile")
print(sftp.stat("/opt/card-system/backend/Dockerfile").st_size)
sftp.close()
cli.close()
print("OK")
