package redis

// refreshSessionTTLLuaScript 刷新Session TTL的Lua脚本
// 功能：
//  1. 检查session key是否存在
//  2. 如果存在，解析JSON，更新last_active_at字段
//  3. 重新设置key，并刷新TTL
//  4. 返回结果（1表示成功，0表示session不存在，-1表示JSON解析失败）
//
// 参数：
//
//	KEYS[1]: session key
//	ARGV[1]: 新的last_active_at时间戳（字符串）
//	ARGV[2]: session过期时间（秒数，字符串）
const refreshSessionTTLLuaScript = `
local sessionKey = KEYS[1]
local lastActiveAt = ARGV[1]
local expireSeconds = tonumber(ARGV[2])

-- 检查session是否存在
local sessionData = redis.call('GET', sessionKey)
if not sessionData then
    return 0
end

-- 解析JSON并更新last_active_at字段
local cjson = require('cjson')
local ok, session = pcall(cjson.decode, sessionData)
if not ok or not session then
    -- JSON解析失败
    return -1
end

-- 更新last_active_at字段（JSON字段名是last_active_at）
session.last_active_at = tonumber(lastActiveAt)

-- 重新编码JSON
local updatedData = cjson.encode(session)

-- 设置更新后的session数据并刷新TTL
redis.call('SET', sessionKey, updatedData, 'EX', expireSeconds)

return 1
`
