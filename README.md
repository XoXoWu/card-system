# 发卡系统（Card System）

自研 App 的卡密（序列号）发卡系统：按月卡 / 季卡 / 年卡等套餐生成与管理限期卡密，App 通过在线验证 API 完成激活、心跳与续费。依据《发卡系统 PRD V1.1》实现，含测试卡发放（激活后 24 小时）。

## 技术栈

| 层 | 选型 |
|---|---|
| 后端 | Go 1.26 + Gin + SQLite（modernc 纯 Go 驱动，无 CGO） |
| 前端 | Vue 3 + Element Plus + ECharts + Vite |
| 客户端 | Lua SDK（card.lua，兼容 Lua 5.1~5.4 / LuaJIT，纯 Lua 实现 HMAC-SHA256） |
| 部署 | Docker Compose（应用容器 + Caddy 自动 HTTPS） |

## 功能

**管理后台**（浏览器访问，JWT 登录）

- 数据看板：总卡数 / 激活率 / 使用中 / 今日激活 / 在线设备（15 分钟）/ 测试卡发放量等指标 + 近 30 天激活趋势 + 套餐分布
- 卡密生成：选套餐批量生成（单批 ≤ 5000），前缀自定义（TEST 为保留前缀），明文仅在当次会话展示，支持复制 / 导出 TXT / CSV
- 测试卡发放：单次 1–10 张，固定 `TEST-XXXX-XXXX-XXXX`，激活后 24 小时到期、限 1 台设备、不计入销售统计、不可作续费卡
- 卡密列表：按末 4 位 / 批次 / 备注 / 完整卡密搜索，按状态 / 类型 / 套餐筛选，详情抽屉展示绑定设备
- 卡密操作：封禁 / 解封 / 作废 / 修改备注 / 解绑设备
- 套餐管理：时长、设备上限（1–10 台）、参考价、启用停用
- 操作日志：登录、生成、封禁、解绑、App 激活 / 续费等全量记录
- 系统设置：修改密码、查看 App 接入凭据（app_id / app_secret）

**验证 API**（App 调用，HMAC-SHA256 签名 + 防重放 + IP 频控）

- `POST /api/v1/activate` 激活：绑定设备、扣减卡密有效期，幂等（同设备重复激活直接成功）
- `POST /api/v1/verify` 心跳：校验状态并更新设备活跃时间，返回剩余天数与离线宽限期
- `POST /api/v1/renew` 续费：新卡时长叠加到当前卡（过期卡从当下起算），续费卡即消耗
- 卡密安全：服务端仅存 SHA-256 哈希与末 4 位；字符集剔除 I/O/0/1；时间戳偏差 > 5 分钟或 nonce 重复均拒绝

## 目录结构

```
card-system/
├── backend/            # Go 后端
│   ├── main.go
│   ├── internal/
│   │   ├── api/        # 路由、中间件（签名/JWT/频控）、验证与管理接口
│   │   ├── auth/       # bcrypt 密码 + JWT
│   │   ├── config/     # 环境变量配置
│   │   ├── db/         # SQLite 建表迁移
│   │   ├── keygen/     # 卡密生成（去混淆字符集）
│   │   └── model/      # 数据模型与响应码
│   ├── cmd/e2e-test/   # 验证 API 端到端测试客户端
│   └── Dockerfile
├── frontend/           # Vue 3 管理后台
├── client/lua/         # Lua 客户端 SDK（card.lua.example 为模板，card.lua 不入库）
├── deploy/             # 服务器部署脚本（secrets.py 不入库，模板见 secrets.py.example）
├── .env.example        # Docker 部署环境变量模板
└── docker-compose.yml  # 应用 + Caddy 编排
```

## 本地开发

```bash
# 后端（数据库 data/data.db）
# 端口要和前端 vite 代理一致（127.0.0.1:18080），开发时设置 CARD_LISTEN：
cd backend
set CARD_LISTEN=:18080      # Windows cmd
go run .
# bash/macOS/linux 写法：CARD_LISTEN=:18080 go run .

# 前端（vite 代理 /api 到 127.0.0.1:18080）
cd frontend
npm install
npm run dev

# 验证 API 端到端测试（需后端先启动）
cd backend && go run ./cmd/e2e-test
```

首次启动自动创建管理员 `admin / admin123`（环境变量可覆盖）与默认套餐月卡 / 季卡 / 年卡，**登录后请立即在系统设置中修改密码**。

## Docker 部署（服务器）

```bash
# 1. 准备环境变量
cp .env.example .env   # 修改 SITE_ADDRESS / CARD_JWT_SECRET / CARD_ADMIN_PASSWORD

# 2. 一条命令拉起（构建前端 + 后端 + Caddy）
docker compose up -d --build

# 3. 数据备份（SQLite 单文件，建议 cron 每日执行）
sqlite3 data/data.db ".backup data/backup-$(date +%F).db"
```

DNS 将域名解析到服务器后，访问 `https://card.example.com` 即为管理后台；App 端 API 同域名（`/api/v1/*`）。数据库与备份都在 `./data/`，迁移服务器时整目录拷走即可。

## App 端接入

1. 管理后台「系统设置」中获取 `app_id` 与 `app_secret`（请勿写入客户端分发）
2. 每次请求携带签名头：

```
X-App-Id:     my-app
X-Timestamp:  1788745872000                       # Unix 毫秒，偏差 ≤ 5 分钟
X-Nonce:      随机不重复字符串                     # 5 分钟窗口内防重放
X-Signature:  HMAC-SHA256(app_secret, timestamp + nonce + body) 的小写十六进制
```

3. 响应统一为 `{"code": 0, "msg": "ok", "data": {...}}`，非 0 见响应码表（1001 卡密不存在 / 1002 已过期 / 1003 已封禁 / 1004 已作废 / 1005 设备数达上限 / 1006 设备未绑定 / 1007 签名错误 / 1008 重放请求）
4. 验证流程：启动时若有本地卡密 → `verify` 心跳（建议每 5–15 分钟）；用户输入新卡 → `activate`；到期换卡 → `renew`
5. `verify` 返回 `grace_hours`（默认 72），App 按此容忍断网；连续超时后引导用户联网验证

## Lua 客户端接入（card.lua）

适合脚本类 App（按键精灵 / 触摸精灵等 Lua 运行环境）：

```bash
cp client/lua/card.lua.example client/lua/card.lua   # 填入 BASE_URL / APP_ID / APP_SECRET
```

```lua
dofile("card.lua")
local ok, msg, days = verify_card("TEST-XXXX-XXXX-XXXX")  -- 激活/验证
local ok, msg, days = heartbeat("TEST-XXXX-XXXX-XXXX")    -- 心跳（建议每 5 分钟）
-- ok: true/false；msg: 失败原因；days: 剩余天数
-- 第 4 个返回值是错误码：1001~1011 卡密问题；-2 网络错误（可进入离线宽限）
```

游戏引擎没有 luasocket 时，注入引擎自己的同步 HTTP：`set_http(function(method, url, headers, body) return status, resp end)`。纯 Lua 实现 SHA-256/HMAC，无外部依赖。

## 测试卡规则（PRD V1.1）

| 项 | 规则 |
|---|---|
| 前缀 / 格式 | 固定 `TEST-XXXX-XXXX-XXXX`，不可自定义 |
| 有效期 | 激活后 24 小时（固定不可配置） |
| 设备上限 | 1 台 |
| 数量 | 单次 1–10 张 |
| 统计 | 不计入销售 KPI 与激活率，看板单独统计 |
| 续费 | 不可作为续费卡（renew 返回 revoked） |
| 管理 | 封禁 / 作废 / 解绑 / 备注与普通卡一致 |

## 安全说明

- **本仓库不含任何真实凭据**：服务器 SSH 信息在 `deploy/secrets.py`（不入库，模板 `secrets.py.example`）；Lua 客户端的真实地址与 app_secret 在 `client/lua/card.lua`（不入库，模板 `card.lua.example`）；部署环境变量在 `.env`（不入库，模板 `.env.example`）。新环境先复制模板填值。
- 上线后**必须**修改默认管理员密码（`CARD_ADMIN_PASSWORD`），并设置强随机的 `CARD_JWT_SECRET`。
- `app_secret` 会随 Lua 客户端一起分发（HMAC 签名方案的前提），它能被逆向提取——这是本方案的已知边界，靠服务端频控 + 防重放 + 卡密哈希存储兜底；对外分发前评估是否可接受。
- `deploy/verify.py` 通过环境变量读取目标与凭据（`CARD_BASE` / `CARD_ADMIN_USER` / `CARD_ADMIN_PASSWORD`），默认打本地 8080。
