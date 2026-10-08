--------------------------------------------------------------------------------
-- example.lua —— card_auth 使用示例
--
-- 场景一：PC / 独立 Lua 环境（自带 luasocket，开箱即用）
-- 场景二：游戏引擎（Cocos2d-x / xlua / LÖVE 等）—— 注入引擎自己的 HTTP
--------------------------------------------------------------------------------

local card_auth = require("card_auth")

--==============================================================================
-- 初始化（App 启动时执行一次）
--==============================================================================
card_auth.init({
    base_url   = "http://your-server-ip",   -- 你的发卡系统地址
    app_id     = "your-app-id",             -- 后台「系统设置」中的 App ID
    app_secret = "your-app-secret",         -- 后台「系统设置」中的 App Secret
    -- 推荐传引擎提供的稳定设备标识，例如：
    -- device_id  = device.getHardwareId(),
    device_info = "Windows 11 · MyApp v1.0.0",
})

--==============================================================================
-- 一、激活卡密（用户输入卡密后调用）
--==============================================================================
local function doActivate(cardKey)
    local r = card_auth.activate(cardKey)
    if r.ok then
        local d = r.data
        print(("激活成功！套餐：%s，剩余 %d 天，到期时间戳 %d"):format(
            d.plan, d.days_left, d.expire_at))
        -- 保存卡密供下次启动时恢复会话
        -- saveToFile("card.key", cardKey)
        return true
    else
        print("激活失败：" .. r.msg .. "（code " .. tostring(r.code) .. "）")
        -- r.code == 1005 表示设备数已满，提示用户在后台解绑旧设备
        return false
    end
end

--==============================================================================
-- 二、心跳验证
--==============================================================================

-- 方式 A：手动单次调用（自己控制时机）
local function checkOnce(cardKey)
    local r = card_auth.verify(cardKey)
    if r.ok then
        print("心跳正常，剩余 " .. r.data.days_left .. " 天")
    elseif card_auth.is_fatal(r.code) then
        print("卡密不可用：" .. r.msg)  -- 过期/封禁/作废/未绑定 → 锁定付费功能
    else
        print("心跳异常：" .. r.msg)     -- 网络/签名等临时问题 → 可给宽限期
    end
    return r
end

-- 方式 B：心跳管理器（推荐）
-- 启动：每 5 分钟自动校验一次；回调里处理 UI 提示与功能开关
card_auth.heartbeat_start(cardKey, 300, function(r, failCount)
    if r.ok then
        -- 心跳成功，刷新剩余天数显示
        print("heartbeat ok, days_left = " .. r.data.days_left)
    elseif card_auth.is_fatal(r.code) then
        -- 卡密过期/被封/作废/设备被解绑 → 立即锁定功能
        print("heartbeat fatal: " .. r.msg)
        -- lockProFeatures()
    elseif failCount >= 3 then
        -- 连续 3 次非致命失败（多为断网）→ 提示离线/进入宽限期
        print("heartbeat lost x" .. failCount .. ", grace mode")
    end
end)

-- 在引擎主循环（update）或每秒定时器中驱动：
-- function onEverySecond() card_auth.heartbeat_tick() end

-- App 退出时：
-- card_auth.heartbeat_stop()

--==============================================================================
-- 三、续费（用户在 App 内输入新卡密叠加时长）
--==============================================================================
local function doRenew(oldKey, newKey)
    local r = card_auth.renew(oldKey, newKey)
    if r.ok then
        print(("续费成功！新到期时间戳 %d（剩余 %d 天）"):format(
            r.data.expire_at, r.data.days_left))
        return true
    end
    print("续费失败：" .. r.msg)
    return false
end

--==============================================================================
-- 四、游戏引擎异步接入（引擎自带异步 HTTP 时使用）
--==============================================================================
--[[
-- 1) 注入引擎的 HTTP 实现（签名要返回 status, body 或 nil, err）
card_auth.set_http_func(function(method, url, headers, body)
    local status, resp = engine.httpSync(method, url, headers, body) -- 同步封装
    return status, resp
end)

-- 2) 完全异步的模式：拿签名好的请求自己发，回包再解析
local req = card_auth.build_request("verify", {
    card_key = savedKey, device_id = myDeviceId,
})
engine.httpAsync(req.method, req.url, req.headers, req.body, function(resp)
    local r = card_auth.parse_response(resp)
    if not r.ok and card_auth.is_fatal(r.code) then lockProFeatures() end
end)
]]

--==============================================================================
-- 附：独立运行本示例的最小流程
--==============================================================================
if arg and arg[1] then
    local cardKey = arg[1]
    if doActivate(cardKey) then
        card_auth.heartbeat_start(cardKey, 300, function() end)
        checkOnce(cardKey)
    end
end

return {
    doActivate = doActivate,
    checkOnce = checkOnce,
    doRenew = doRenew,
}
