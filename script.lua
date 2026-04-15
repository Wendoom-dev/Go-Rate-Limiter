-- KEYS[1] = rate_limit:{api_key}
-- ARGV[1] = max_tokens (e.g. 10)
-- ARGV[2] = refill_rate (tokens per second, e.g. 1)
-- ARGV[3] = current_time (in milliseconds)
-- all this is sent by go program

local key = KEYS[1]

local max_tokens = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Fetch current state
local tokens = tonumber(redis.call("HGET", key, "tokens"))
local last_ts = tonumber(redis.call("HGET", key, "ts"))

-- If first time, initialize
if tokens == nil then
    tokens = max_tokens
    last_ts = now
end

-- Calculate time passed (in seconds)
local delta_ms = now - last_ts
local delta_seconds = delta_ms / 1000.0

-- Refill tokens
tokens = tokens + (delta_seconds * refill_rate)
if tokens > max_tokens then
    tokens = max_tokens
end

local allowed = 0
local retry_after_ms = 0

-- Check if request allowed
if tokens >= 1 then
    allowed = 1
    tokens = tokens - 1
else
    allowed = 0
    local needed = 1 - tokens
    retry_after_ms = (needed / refill_rate) * 1000
end

-- Update state
redis.call("HSET", key, "tokens", tokens)
redis.call("HSET", key, "ts", now)
redis.call("EXPIRE", key, 30)

-- Return:
-- {allowed (0/1), tokens_left, retry_after_ms}
return {allowed, tokens, retry_after_ms}