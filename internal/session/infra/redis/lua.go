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

// storeSessionLuaScript 存储Session的Lua脚本（原子性操作）
// 功能：
//  1. 设置session数据
//  2. 将device_id添加到用户会话集合
//  3. 设置集合过期时间
//  4. 返回结果（1表示成功）
//
// 参数：
//
//	KEYS[1]: session key
//	KEYS[2]: user sessions set key
//	ARGV[1]: session数据（JSON字符串）
//	ARGV[2]: device_id
//	ARGV[3]: session过期时间（秒数，字符串）
const storeSessionLuaScript = `
local sessionKey = KEYS[1]
local setKey = KEYS[2]
local sessionData = ARGV[1]
local deviceId = ARGV[2]
local expireSeconds = tonumber(ARGV[3])

-- 设置session数据
redis.call('SET', sessionKey, sessionData, 'EX', expireSeconds)

-- 将device_id添加到集合
redis.call('SADD', setKey, deviceId)

-- 设置集合过期时间
redis.call('EXPIRE', setKey, expireSeconds)

return 1
`

// getSessionsByUserIDLuaScript 获取用户所有会话的Lua脚本（原子性操作）
// 功能：
//  1. 从集合中获取所有device_id
//  2. 批量获取所有session数据
//  3. 过滤掉不存在的session，并从集合中移除
//  4. 返回所有有效的session数据数组
//
// 参数：
//
//	KEYS[1]: user sessions set key
//	ARGV[1]: user_id（用于构建session key）
const getSessionsByUserIDLuaScript = `
local setKey = KEYS[1]
local userId = ARGV[1]

-- 从集合中获取所有device_id
local deviceIds = redis.call('SMEMBERS', setKey)
if not deviceIds or #deviceIds == 0 then
    return {}
end

local sessions = {}
local expiredDevices = {}

-- 批量获取所有session
for i = 1, #deviceIds do
    local deviceId = deviceIds[i]
    -- 构建session key，格式: kim:user:session:{user_id}:device_id
    local sessionKey = 'kim:user:session:{' .. userId .. '}:' .. deviceId
    local sessionData = redis.call('GET', sessionKey)
    
    if sessionData then
        table.insert(sessions, sessionData)
    else
        -- session已过期，记录需要从集合中移除的device_id
        table.insert(expiredDevices, deviceId)
    end
end

-- 清理过期的device_id
if #expiredDevices > 0 then
    for i = 1, #expiredDevices do
        redis.call('SREM', setKey, expiredDevices[i])
    end
end

return sessions
`

// deleteSessionLuaScript 删除会话的Lua脚本（原子性操作）
// 功能：
//  1. 删除session数据
//  2. 从用户会话集合中移除device_id
//  3. 返回结果（1表示成功，0表示session不存在）
//
// 参数：
//
//	KEYS[1]: session key
//	KEYS[2]: user sessions set key
//	ARGV[1]: device_id
const deleteSessionLuaScript = `
local sessionKey = KEYS[1]
local setKey = KEYS[2]
local deviceId = ARGV[1]

-- 检查session是否存在
local exists = redis.call('EXISTS', sessionKey)
if exists == 0 then
    return 0
end

-- 删除session数据
redis.call('DEL', sessionKey)

-- 从集合中移除device_id
redis.call('SREM', setKey, deviceId)

return 1
`

// deleteSessionsByUserIDLuaScript 删除用户所有会话的Lua脚本（原子性操作）
// 功能：
//  1. 从集合中获取所有device_id
//  2. 批量删除所有session数据
//  3. 删除集合
//  4. 返回删除的session数量
//
// 参数：
//
//	KEYS[1]: user sessions set key
//	ARGV[1]: user_id（用于构建session key）
const deleteSessionsByUserIDLuaScript = `
local setKey = KEYS[1]
local userId = ARGV[1]

-- 从集合中获取所有device_id
local deviceIds = redis.call('SMEMBERS', setKey)
if not deviceIds or #deviceIds == 0 then
    -- 删除集合（即使为空也删除）
    redis.call('DEL', setKey)
    return 0
end

local deletedCount = 0

-- 批量删除所有session
for i = 1, #deviceIds do
    local deviceId = deviceIds[i]
    -- 构建session key，格式: kim:user:session:{user_id}:device_id
    local sessionKey = 'kim:user:session:{' .. userId .. '}:' .. deviceId
    local deleted = redis.call('DEL', sessionKey)
    if deleted > 0 then
        deletedCount = deletedCount + 1
    end
end

-- 删除集合
redis.call('DEL', setKey)

return deletedCount
`
