--------------------------------------------------------------------------------
-- card_auth.lua —— 发卡系统 App 端验证模块（纯 Lua，无第三方依赖）
--
-- 兼容 Lua 5.1 / 5.2 / 5.3 / 5.4 / LuaJIT（Cocos2d-x、xlua、LÖVE 等引擎均可直接使用）
--
-- 功能：
--   1. 卡密激活（activate）      —— 用户输入卡密后首次调用，绑定设备并开始计时
--   2. 心跳验证（verify）         —— App 运行期间定期调用，实时校验卡密状态
--   3. 心跳管理器                 —— 自动按间隔执行心跳，通过 update() 驱动
--   4. 续费（renew）              —— 用新卡密叠加时长
--
-- 请求签名机制（与服务端约定）：
--   X-App-Id    : 应用 ID（后台「设置」中配置）
--   X-Timestamp : 毫秒时间戳（与服务器时差须在 ±5 分钟内）
--   X-Nonce     : 每次请求唯一的随机串（防重放）
--   X-Signature : hex(HMAC-SHA256(app_secret, timestamp .. nonce .. body))
--
-- 使用示例见 example.lua
--------------------------------------------------------------------------------

local card_auth = { _VERSION = "card_auth 1.0" }

--------------------------------------------------------------------------------
-- 一、32 位位运算（纯 Lua 实现，不依赖 bit / bit32 库）
--------------------------------------------------------------------------------

local function band(a, b)
    local r, p = 0, 1
    for _ = 1, 32 do
        if a % 2 == 1 and b % 2 == 1 then r = r + p end
        a = math.floor(a / 2)
        b = math.floor(b / 2)
        p = p * 2
    end
    return r
end

local function bxor(a, b)
    local r, p = 0, 1
    for _ = 1, 32 do
        if (a % 2 == 1) ~= (b % 2 == 1) then r = r + p end
        a = math.floor(a / 2)
        b = math.floor(b / 2)
        p = p * 2
    end
    return r
end

local function bnot(a)
    return 0xFFFFFFFF - a
end

local function rshift(a, n)
    return math.floor(a / 2 ^ n)
end

local function rrotate(a, n)
    return math.floor(a / 2 ^ n) + (a % 2 ^ n) * 2 ^ (32 - n)
end

--------------------------------------------------------------------------------
-- 二、SHA-256 与 HMAC-SHA256（纯 Lua 实现）
--------------------------------------------------------------------------------

local K = {
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1,
    0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786,
    0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147,
    0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
    0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a,
    0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}

local hex_byte = {}
for i = 0, 255 do
    hex_byte[i] = string.format("%02x", i)
end

-- 返回 32 字节二进制摘要
local function sha256_bin(msg)
    local len = #msg
    local padded = msg .. "\128" .. string.rep("\0", (55 - len) % 64)
    local bitlen = len * 8
    for i = 7, 0, -1 do
        padded = padded .. string.char(math.floor(bitlen / 2 ^ (8 * i)) % 256)
    end

    local h1, h2, h3, h4 = 0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a
    local h5, h6, h7, h8 = 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19

    local W = {}
    for off = 1, #padded, 64 do
        for j = 0, 15 do
            local p = off + j * 4
            W[j] = string.byte(padded, p) * 0x1000000
                + string.byte(padded, p + 1) * 0x10000
                + string.byte(padded, p + 2) * 0x100
                + string.byte(padded, p + 3)
        end
        for j = 16, 63 do
            local x, y = W[j - 15], W[j - 2]
            local s0 = bxor(bxor(rrotate(x, 7), rrotate(x, 18)), rshift(x, 3))
            local s1 = bxor(bxor(rrotate(y, 17), rrotate(y, 19)), rshift(y, 10))
            W[j] = (W[j - 16] + s0 + W[j - 7] + s1) % 0x100000000
        end

        local a, b, c, d = h1, h2, h3, h4
        local e, f, g, h = h5, h6, h7, h8

        for j = 0, 63 do
            local S1 = bxor(bxor(rrotate(e, 6), rrotate(e, 11)), rrotate(e, 25))
            local ch = bxor(band(e, f), band(bnot(e), g))
            local t1 = (h + S1 + ch + K[j + 1] + W[j]) % 0x100000000
            local S0 = bxor(bxor(rrotate(a, 2), rrotate(a, 13)), rrotate(a, 22))
            local maj = bxor(bxor(band(a, b), band(a, c)), band(b, c))
            local t2 = (S0 + maj) % 0x100000000
            h, g, f = g, f, e
            e = (d + t1) % 0x100000000
            d, c, b = c, b, a
            a = (t1 + t2) % 0x100000000
        end

        h1 = (h1 + a) % 0x100000000
        h2 = (h2 + b) % 0x100000000
        h3 = (h3 + c) % 0x100000000
        h4 = (h4 + d) % 0x100000000
        h5 = (h5 + e) % 0x100000000
        h6 = (h6 + f) % 0x100000000
        h7 = (h7 + g) % 0x100000000
        h8 = (h8 + h) % 0x100000000
    end

    local out = {}
    local hs = { h1, h2, h3, h4, h5, h6, h7, h8 }
    for i = 1, 8 do
        local v = hs[i]
        out[#out + 1] = string.char(
            math.floor(v / 0x1000000) % 256,
            math.floor(v / 0x10000) % 256,
            math.floor(v / 0x100) % 256,
            v % 256)
    end
    return table.concat(out)
end

local function to_hex(bin)
    local t = {}
    for i = 1, #bin do
        t[i] = hex_byte[string.byte(bin, i)]
    end
    return table.concat(t)
end

function card_auth.sha256_hex(msg)
    return to_hex(sha256_bin(msg))
end

-- HMAC-SHA256，返回小写 hex 字符串
function card_auth.hmac_sha256_hex(key, msg)
    if #key > 64 then
        key = sha256_bin(key)
    end
    local ipad, opad = {}, {}
    for i = 1, 64 do
        local k = i <= #key and string.byte(key, i) or 0
        ipad[i] = string.char(bxor(k, 0x36))
        opad[i] = string.char(bxor(k, 0x5c))
    end
    local inner = sha256_bin(table.concat(ipad) .. msg)
    return to_hex(sha256_bin(table.concat(opad) .. inner))
end

--------------------------------------------------------------------------------
-- 三、轻量 JSON（编码请求 / 解析响应）
--------------------------------------------------------------------------------

-- 大整数转十进制字符串（Lua 5.1/5.2 的 string.format("%d") 只有 32 位，会溢出）
local function int_to_str(n)
    n = math.floor(n)
    if n < 0 then
        return "-" .. int_to_str(-n)
    end
    local high = math.floor(n / 1000000000)
    local low = n % 1000000000
    if high == 0 then
        return string.format("%d", low)
    end
    return string.format("%d%09d", high, low)
end

-- 当前毫秒时间戳字符串（tostring 在 Lua 5.1/LuaJIT 下会把大数变成科学计数法）
local function ms_timestamp()
    return int_to_str(os.time() * 1000)
end

--------------------------------------------------------------------------------
-- 三、轻量 JSON（编码请求 / 解析响应）
--------------------------------------------------------------------------------

local function json_escape(s)
    return (string.gsub(s, '[%c"\\]', function(c)
        if c == '"' then return '\\"' end
        if c == '\\' then return '\\\\' end
        if c == '\n' then return '\\n' end
        if c == '\t' then return '\\t' end
        if c == '\r' then return '\\r' end
        return string.format('\\u%04x', string.byte(c))
    end))
end

local function json_encode(v)
    local t = type(v)
    if t == "string" then
        return '"' .. json_escape(v) .. '"'
    elseif t == "number" then
        if v % 1 == 0 and math.abs(v) < 1e15 then
            return int_to_str(v)
        end
        return string.format("%.14g", v)
    elseif t == "boolean" then
        return tostring(v)
    elseif t == "table" then
        if #v > 0 then
            local parts = {}
            for i = 1, #v do parts[i] = json_encode(v[i]) end
            return "[" .. table.concat(parts, ",") .. "]"
        end
        local parts = {}
        for k, val in pairs(v) do
            parts[#parts + 1] = '"' .. json_escape(tostring(k)) .. '":' .. json_encode(val)
        end
        return "{" .. table.concat(parts, ",") .. "}"
    end
    error("json_encode: 无法编码类型 " .. t)
end

local function json_decode(s)
    local pos = 1
    local len = #s

    local function skip_ws()
        while pos <= len and s:sub(pos, pos):find("%s") do
            pos = pos + 1
        end
    end

    local function utf8_char(cp)
        if cp < 0x80 then
            return string.char(cp)
        elseif cp < 0x800 then
            return string.char(0xC0 + math.floor(cp / 64), 0x80 + cp % 64)
        elseif cp < 0x10000 then
            return string.char(
                0xE0 + math.floor(cp / 4096),
                0x80 + math.floor(cp / 64) % 64,
                0x80 + cp % 64)
        end
        return string.char(
            0xF0 + math.floor(cp / 262144),
            0x80 + math.floor(cp / 4096) % 64,
            0x80 + math.floor(cp / 64) % 64,
            0x80 + cp % 64)
    end

    local function parse_string()
        pos = pos + 1 -- 跳过开头引号
        local buf = {}
        while true do
            local ch = s:sub(pos, pos)
            if ch == "" then error("json: 字符串未闭合") end
            if ch == '"' then
                pos = pos + 1
                return table.concat(buf)
            end
            if ch == "\\" then
                local e = s:sub(pos + 1, pos + 1)
                pos = pos + 2
                if e == "n" then buf[#buf + 1] = "\n"
                elseif e == "t" then buf[#buf + 1] = "\t"
                elseif e == "r" then buf[#buf + 1] = "\r"
                elseif e == "b" then buf[#buf + 1] = "\b"
                elseif e == "f" then buf[#buf + 1] = "\f"
                elseif e == "u" then
                    local cp = tonumber(s:sub(pos, pos + 3), 16)
                    if not cp then error("json: 非法的 \\u 转义") end
                    pos = pos + 4
                    if cp >= 0xD800 and cp < 0xDC00 then
                        -- 高位代理项，尝试合并低位代理项
                        if s:sub(pos, pos + 1) == "\\u" then
                            local lo = tonumber(s:sub(pos + 2, pos + 5), 16)
                            if lo and lo >= 0xDC00 and lo < 0xE000 then
                                pos = pos + 6
                                cp = 0x10000 + (cp - 0xD800) * 0x400 + (lo - 0xDC00)
                            end
                        end
                    end
                    buf[#buf + 1] = utf8_char(cp)
                else
                    buf[#buf + 1] = e
                end
            else
                buf[#buf + 1] = ch
                pos = pos + 1
            end
        end
    end

    local function parse_value()
        skip_ws()
        local c = s:sub(pos, pos)
        if c == "{" then
            pos = pos + 1
            local obj = {}
            skip_ws()
            if s:sub(pos, pos) == "}" then pos = pos + 1 return obj end
            while true do
                skip_ws()
                local k = parse_string()
                skip_ws()
                pos = pos + 1 -- 跳过 ':'
                obj[k] = parse_value()
                skip_ws()
                local d = s:sub(pos, pos)
                if d == "," then pos = pos + 1
                elseif d == "}" then pos = pos + 1 return obj
                else error("json: 对象语法错误 @" .. pos) end
            end
        elseif c == "[" then
            pos = pos + 1
            local arr = {}
            skip_ws()
            if s:sub(pos, pos) == "]" then pos = pos + 1 return arr end
            while true do
                arr[#arr + 1] = parse_value()
                skip_ws()
                local d = s:sub(pos, pos)
                if d == "," then pos = pos + 1
                elseif d == "]" then pos = pos + 1 return arr
                else error("json: 数组语法错误 @" .. pos) end
            end
        elseif c == '"' then
            return parse_string()
        elseif s:sub(pos, pos + 3) == "true" then
            pos = pos + 4
            return true
        elseif s:sub(pos, pos + 4) == "false" then
            pos = pos + 5
            return false
        elseif s:sub(pos, pos + 3) == "null" then
            pos = pos + 4
            return nil
        else
            local num = s:match("^%-?%d+%.?%d*", pos)
            if not num then error("json: 非法字符 @" .. pos) end
            local exp = s:sub(pos + #num):match("^[eE][%+%-]?%d+")
            if exp then num = num .. exp end
            pos = pos + #num
            return tonumber(num)
        end
    end

    return parse_value()
end

--------------------------------------------------------------------------------
-- 四、配置 / HTTP 适配
--------------------------------------------------------------------------------

local cfg = {
    base_url   = "http://127.0.0.1",     -- 发卡系统地址
    app_id     = "my-app",               -- 后台「设置」里的 App ID
    app_secret = "dev-secret-change-me", -- 后台「设置」里的 App Secret（切勿内置到客户端分发包中明文存放，混淆/加密处理）
}

local device_id
local device_info
local http_func -- fn(method, url, headers_tbl, body_str) -> status_num, resp_body | nil, err_str

-- 默认 HTTP 实现：luasocket（PC / 独立 Lua 环境可用；游戏引擎请用 set_http_func 注入）
local function default_http(method, url, headers, body)
    local ok1, http = pcall(require, "socket.http")
    local ok2, ltn12 = pcall(require, "ltn12")
    if not ok1 or not ok2 then
        return nil, "未找到 luasocket，请调用 card_auth.set_http_func() 注入 HTTP 实现"
    end
    local resp = {}
    local result, code = http.request({
        url = url,
        method = method,
        headers = headers,
        source = body and ltn12.source.string(body) or nil,
        sink = ltn12.sink.table(resp),
    })
    if result ~= 1 then
        return nil, tostring(code)
    end
    return code, table.concat(resp)
end

local function http_do(method, url, headers, body)
    local fn = http_func or default_http
    local ok, status, resp = pcall(fn, method, url, headers, body)
    if not ok then
        return nil, tostring(status)
    end
    return status, resp
end

function card_auth.set_http_func(fn)
    http_func = fn
end

-- 设备 ID 兜底生成（优先在 init 中显式传入引擎的设备标识）
local function default_device_id()
    local path = (os.getenv("TEMP") or os.getenv("TMP") or "/tmp") .. "/card_auth_device_id"
    local f = io.open(path, "r")
    if f then
        local id = f:read("*l")
        f:close()
        if id and #id > 8 then return id end
    end
    local id = string.format("dev-%d-%d", os.time(), math.random(10000000, 99999999))
    local w = io.open(path, "w")
    if w then w:write(id) w:close() end
    return id
end

--------------------------------------------------------------------------------
-- 五、错误码表（与服务端约定，见 PRD 7.5）
--------------------------------------------------------------------------------

local ERR_MSG = {
    [0]    = "成功",
    [1001] = "无效卡密",
    [1002] = "卡密已过期",
    [1003] = "卡密已被封禁",
    [1004] = "卡密已作废",
    [1005] = "绑定设备数已达上限",
    [1006] = "当前设备未绑定该卡密",
    [1007] = "签名校验失败",
    [1008] = "重复请求（防重放）",
    [1009] = "请求格式错误",
    [1011] = "卡密尚未激活",
    [1020] = "请求过于频繁",
}

-- 这些错误码意味着卡密已不可用，App 应锁定付费功能
local FATAL_CODES = {
    [1001] = true, [1002] = true, [1003] = true,
    [1004] = true, [1006] = true, [1011] = true,
}

function card_auth.is_fatal(code)
    return FATAL_CODES[code] == true
end

--------------------------------------------------------------------------------
-- 六、核心请求
--------------------------------------------------------------------------------

local nonce_seq = 0

local function gen_nonce()
    nonce_seq = nonce_seq + 1
    return string.format("n-%d-%d-%d-%d",
        os.time(), nonce_seq, math.random(0, 999999), math.floor(os.clock() * 1000000))
end

-- 构造已签名的请求（供异步引擎自行发送：url/headers/body 拿到后用自己的 HTTP 发出）
function card_auth.build_request(endpoint, payload)
    local body = json_encode(payload)
    local ts = ms_timestamp()
    local nonce = gen_nonce()
    local sig = card_auth.hmac_sha256_hex(cfg.app_secret, ts .. nonce .. body)
    return {
        method = "POST",
        url = cfg.base_url .. "/api/v1/" .. endpoint,
        headers = {
            ["Content-Type"] = "application/json",
            ["Content-Length"] = tostring(#body),
            ["X-App-Id"] = cfg.app_id,
            ["X-Timestamp"] = ts,
            ["X-Nonce"] = nonce,
            ["X-Signature"] = sig,
        },
        body = body,
    }
end

-- 解析服务端响应（异步引擎拿到响应体后调这个；同步场景无需直接使用）
function card_auth.parse_response(raw)
    local ok, decoded = pcall(json_decode, raw or "")
    if not ok or type(decoded) ~= "table" then
        return { ok = false, code = -1, msg = "响应解析失败", raw = raw }
    end
    local code = decoded.code
    if type(code) ~= "number" then code = -1 end
    return {
        ok = code == 0,
        code = code,
        msg = ERR_MSG[code] or "未知错误",
        data = decoded.data,
    }
end

local function signed_request(endpoint, payload)
    local req = card_auth.build_request(endpoint, payload)
    local status, resp = http_do(req.method, req.url, req.headers, req.body)
    if not status then
        return { ok = false, code = -2, msg = "网络错误：" .. tostring(resp) }
    end
    if status ~= 200 then
        return { ok = false, code = -3, msg = "HTTP " .. tostring(status) }
    end
    return card_auth.parse_response(resp)
end

--------------------------------------------------------------------------------
-- 七、对外 API
--------------------------------------------------------------------------------

-- 初始化
-- opts = {
--   base_url   = "http://your-server-ip",
--   app_id     = "my-app",
--   app_secret = "your-app-secret",
--   device_id  = "引擎提供的稳定设备标识（推荐）",
--   device_info= "Windows 11 · MyApp 1.0",
-- }
function card_auth.init(opts)
    opts = opts or {}
    if opts.base_url then
        cfg.base_url = tostring(opts.base_url):gsub("/+$", "")
    end
    if opts.app_id then cfg.app_id = opts.app_id end
    if opts.app_secret then cfg.app_secret = opts.app_secret end
    device_id = opts.device_id or device_id or default_device_id()
    device_info = opts.device_info or device_info or "unknown"
end

function card_auth.get_device_id()
    return device_id
end

-- 激活卡密：用户输入卡密后调用（同一设备重复激活幂等，返回相同结果）
-- 返回 { ok, code, msg, data = { card_tail, plan, is_test, expire_at, days_left, devices={bound,max}, grace_hours } }
function card_auth.activate(card_key)
    return signed_request("activate", {
        card_key = card_key,
        device_id = device_id,
        device_info = device_info,
    })
end

-- 心跳验证：App 运行期间定期调用，返回结构同 activate
function card_auth.verify(card_key)
    return signed_request("verify", {
        card_key = card_key,
        device_id = device_id,
    })
end

-- 续费：用一张未使用的新卡给当前卡叠加时长（测试卡不能作为续费卡）
function card_auth.renew(card_key, new_card_key)
    return signed_request("renew", {
        current_card_key = card_key,
        new_card_key = new_card_key,
        device_id = device_id,
    })
end

--------------------------------------------------------------------------------
-- 八、心跳管理器
-- 用法：
--   card_auth.heartbeat_start(saved_card_key, 300, function(result) ... end)
--   然后在引擎主循环或定时器里每帧/每秒调用 card_auth.heartbeat_tick()
--   退出时调用 card_auth.heartbeat_stop()
--------------------------------------------------------------------------------

local hb = {
    running = false,
    card_key = nil,
    interval = 300,
    next_at = 0,
    last_result = nil,
    fail_count = 0,
}

function card_auth.heartbeat_start(card_key, interval_sec, on_result)
    hb.card_key = card_key
    hb.interval = interval_sec or 300
    hb.running = true
    hb.next_at = 0          -- 立即执行第一次心跳
    hb.last_result = nil
    hb.fail_count = 0
    hb.on_result = on_result
end

-- 由宿主程序周期性调用（例如每秒一次）；到点后自动执行一次 verify
function card_auth.heartbeat_tick()
    if not hb.running or not hb.card_key then return end
    local now = os.time()
    if now < hb.next_at then return end
    hb.next_at = now + hb.interval -- 先占位，防止重入

    local r = card_auth.verify(hb.card_key)
    hb.last_result = r
    if r.ok then
        hb.fail_count = 0
    else
        hb.fail_count = hb.fail_count + 1
    end
    if hb.on_result then
        pcall(hb.on_result, r, hb.fail_count)
    end
    return r
end

function card_auth.heartbeat_stop()
    hb.running = false
end

function card_auth.heartbeat_info()
    return {
        running = hb.running,
        interval = hb.interval,
        fail_count = hb.fail_count,
        last_result = hb.last_result,
    }
end

return card_auth
