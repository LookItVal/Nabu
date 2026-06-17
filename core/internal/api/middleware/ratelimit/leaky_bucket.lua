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
    -- First request from this IP: start with an empty bucket and set last leak time to now
    tokens    = 0
    last_leak = now
    redis.call("HMSET", key, "tokens", tokens, "last_leak_ms", last_leak)
    -- TTL slightly longer than it would take to fully refill, so idle keys expire cleanly
    redis.call("PEXPIRE", key, math.ceil(capacity / leak_rate) * 1000 + 1000)
    return {1, tokens, capacity}  -- allowed, remaining, capacity
end

-- Calculate how many tokens have leaked since the last request
local elapsed_seconds = (now - last_leak) / 1000
local leaked = math.floor(elapsed_seconds * leak_rate)

if leaked > 0 then
    tokens    = tokens + leaked
    last_leak = last_leak + math.floor(leaked / leak_rate) * 1000
end

if tokens > capacity then
    -- Bucket is full (no room), reject
    redis.call("HMSET", key, "tokens", tokens, "last_leak_ms", last_leak)
    redis.call("PEXPIRE", key, math.ceil(capacity / leak_rate) * 1000 + 1000)
    -- Return: denied, remaining (0), capacity, ms until next token leaks
    local ms_until_next = math.ceil((1 / leak_rate) * 1000) - (now - last_leak)
    return {0, 0, capacity, math.max(0, ms_until_next)}
end

-- Allow the request, add one token
tokens = tokens + 1
redis.call("HMSET", key, "tokens", tokens, "last_leak_ms", last_leak)
redis.call("PEXPIRE", key, math.ceil(capacity / leak_rate) * 1000 + 1000)
return {1, tokens, capacity}