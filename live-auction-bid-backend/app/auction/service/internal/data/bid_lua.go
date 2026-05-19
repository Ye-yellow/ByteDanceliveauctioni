package data

const AtomicBidLua = `
-- KEYS[1] lot hash key
-- KEYS[2] bid stream key
-- KEYS[3] ranking zset key
-- ARGV[1] bid id
-- ARGV[2] user id
-- ARGV[3] nickname
-- ARGV[4] amount
-- ARGV[5] min increment
-- ARGV[6] now ms
-- ARGV[7] anti-snipe extend ms
--
-- Intended semantics:
-- 1. read current_price/version/status/ends_at
-- 2. reject stale/ended/non-live bids
-- 3. validate amount >= current_price + min_increment
-- 4. update current_price/winner/version/ends_at if needed
-- 5. append bid event to stream
-- 6. update ranking zset
-- 7. return authoritative lot snapshot fields
return {err = 'AtomicBidLua is a design placeholder'}
`
