#!/bin/bash
set -e
cd /opt/card-system
# 上传的新 Dockerfile 已由 push 脚本写入，这里直接重建
docker compose up -d --build 2>&1 | tail -15
echo "===DONE==="
sleep 4
docker compose ps --format '{{.Name}} {{.Status}}'
echo "---index---"
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1/
curl -s http://127.0.0.1/ | head -c 300
