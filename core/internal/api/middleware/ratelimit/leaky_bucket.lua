-- Leaky bucket rate limiter.
--
-- KEYS[1]  = Redis key for this IP (e.g. "ratelimit:ip:127.0.0.1")
-- ARGV[1]  = bucket capacity  (max tokens, e.g. 10)
-- ARGV[2]  = leak rate        (tokens leaked per second, e.g. 1)
-- ARGV[3]  = current Unix timestamp in milliseconds

local key       = KEYS[1]
local capacity  = tonumber(ARGV[1])
local leak_rate = tonumber(ARGV[2])  -- tokens per second
local now       = tonumber(ARGV[3])  -- milliseconds

-- Read existing bucket state: {tokens, last_leak_time_ms}
local data = redis.call("HMGET", key, "tokens", "last_leak_ms")
local tokens     = tonumber(data[1])
local last_leak  = tonumber(data[2])

if tokens == nil then
    -- First request from this IP: initialize the bucket state at the current time.
    -- "tokens" represents current bucket fill (used capacity), not remaining capacity.
    tokens    = 0
    last_leak = now
end

-- Calculate how many tokens have leaked since the last request
local elapsed_seconds = (now - last_leak) / 1000
local leaked = math.floor(elapsed_seconds * leak_rate)

if leaked > 0 then
    tokens    = math.max(0, tokens - leaked)
    last_leak = last_leak + math.floor((leaked / leak_rate) * 1000)
end

-- Add one token for the current request after applying leakage.
tokens = tokens + 1

if tokens > capacity then
    -- Bucket is full, reject without recording the over-capacity token.
    redis.call("HMSET", key, "tokens", tokens, "last_leak_ms", last_leak)
    redis.call("PEXPIRE", key, math.ceil(capacity / leak_rate) * 1000 + 1000)
    -- Return: denied, current bucket fill, capacity, ms until next token leaks.
    local ms_until_next = math.ceil((1 / leak_rate) * 1000) - (now - last_leak)
    return {0, tokens, capacity, math.max(0, ms_until_next)}
end

-- Allow the request and persist the updated bucket fill.
redis.call("HMSET", key, "tokens", tokens, "last_leak_ms", last_leak)
redis.call("PEXPIRE", key, math.ceil(capacity / leak_rate) * 1000 + 1000)
return {1, tokens, capacity}