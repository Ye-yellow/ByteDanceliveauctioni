package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

const (
	runtimeIdempotencyTTL = 24 * time.Hour
	runtimeStateTTL       = 7 * 24 * time.Hour
	runtimeRecentLimit    = int64(20)
)

var runtimePlaceBidScript = redis.NewScript(`
local state_key = KEYS[1]
local ranking_key = KEYS[2]
local rankmeta_key = KEYS[3]
local participants_key = KEYS[4]
local recent_key = KEYS[5]
local idem_key = KEYS[6]

local bid_id = ARGV[1]
local user_id = ARGV[2]
local nickname = ARGV[3]
local amount = tonumber(ARGV[4])
local currency = ARGV[5]
local now_ms = tonumber(ARGV[6])
local idem_ttl = tonumber(ARGV[7])
local recent_limit = tonumber(ARGV[8])
local status_live = tonumber(ARGV[9])
local status_extended = tonumber(ARGV[10])
local status_settled = tonumber(ARGV[11])
local stage_bidding = tonumber(ARGV[12])
local stage_duel = tonumber(ARGV[13])
local stage_settle = tonumber(ARGV[14])

local replay = redis.call('GET', idem_key)
if replay then
  local replay_obj = cjson.decode(replay)
  replay_obj.replayed = true
  return cjson.encode(replay_obj)
end

local lot_id = redis.call('HGET', state_key, 'lot_id')
if not lot_id then
  return cjson.encode({ ok = false, message = 'lot runtime state is missing' })
end

local status = tonumber(redis.call('HGET', state_key, 'status') or '0')
if status ~= status_live and status ~= status_extended then
  return cjson.encode({ ok = false, message = 'lot is not live' })
end

local ends_at = tonumber(redis.call('HGET', state_key, 'ends_at_unix_ms') or '0')
if ends_at > 0 and now_ms > ends_at then
  return cjson.encode({ ok = false, message = 'auction has ended' })
end

local current_amount = tonumber(redis.call('HGET', state_key, 'current_amount') or '0')
local current_currency = redis.call('HGET', state_key, 'current_currency') or ''
local min_increment_amount = tonumber(redis.call('HGET', state_key, 'min_increment_amount') or '0')
if currency == '' then
  return cjson.encode({ ok = false, message = 'bid amount and currency are required' })
end
if currency ~= current_currency then
  return cjson.encode({ ok = false, message = 'bid currency must match lot currency' })
end

local previous_leader_id = redis.call('HGET', state_key, 'leading_user_id') or ''
if previous_leader_id ~= '' and previous_leader_id == user_id then
  return cjson.encode({ ok = false, message = '你当前已经是最高价，等其他人出价后再加价' })
end
if amount < current_amount + min_increment_amount then
  return cjson.encode({ ok = false, message = 'bid amount is lower than current price plus min increment' })
end

local room_id = redis.call('HGET', state_key, 'room_id') or ''
local version = tonumber(redis.call('HGET', state_key, 'version') or '0') + 1
local ends_before_bid = ends_at
local extend_count = tonumber(redis.call('HGET', state_key, 'duel_extend_count') or '0')
local extend_count_before = extend_count
local playbook_stage = stage_bidding
local settled_at = tonumber(redis.call('HGET', state_key, 'settled_at_unix_ms') or '0')
local winner_user_id = redis.call('HGET', state_key, 'winner_user_id') or ''
local winner_nickname = redis.call('HGET', state_key, 'winner_nickname') or ''
local final_amount = tonumber(redis.call('HGET', state_key, 'final_amount') or '0')
local final_currency = redis.call('HGET', state_key, 'final_currency') or currency
local duel_active = tonumber(redis.call('HGET', state_key, 'duel_active') or '0')
local duel_user_a_id = redis.call('HGET', state_key, 'duel_user_a_id') or ''
local duel_user_a_nickname = redis.call('HGET', state_key, 'duel_user_a_nickname') or ''
local duel_user_b_id = redis.call('HGET', state_key, 'duel_user_b_id') or ''
local duel_user_b_nickname = redis.call('HGET', state_key, 'duel_user_b_nickname') or ''
local duel_started_at = tonumber(redis.call('HGET', state_key, 'duel_started_at_unix_ms') or '0')
local duel_ends_at = tonumber(redis.call('HGET', state_key, 'duel_ends_at_unix_ms') or '0')

local cap_amount_raw = redis.call('HGET', state_key, 'cap_amount')
local cap_amount = tonumber(cap_amount_raw or '0')
local cap_currency = redis.call('HGET', state_key, 'cap_currency') or ''
if cap_amount > 0 then
  if currency ~= cap_currency then
    return cjson.encode({ ok = false, message = 'bid currency must match cap price currency' })
  end
  if amount >= cap_amount then
    status = status_settled
    settled_at = now_ms
    winner_user_id = user_id
    winner_nickname = nickname
    final_amount = amount
    final_currency = currency
    playbook_stage = stage_settle
    duel_active = 0
  end
end

if status ~= status_settled then
  local anti_snipe_window_seconds = tonumber(redis.call('HGET', state_key, 'anti_snipe_window_seconds') or '0')
  local anti_snipe_extend_seconds = tonumber(redis.call('HGET', state_key, 'anti_snipe_extend_seconds') or '0')
  local max_extend_count = tonumber(redis.call('HGET', state_key, 'max_extend_count') or '0')
  local remaining_ms = ends_at - now_ms
  if remaining_ms > 0 and remaining_ms <= anti_snipe_window_seconds * 1000 and extend_count < max_extend_count then
    ends_at = ends_at + anti_snipe_extend_seconds * 1000
    extend_count = extend_count + 1
    status = status_extended
    duel_ends_at = ends_at
  end
end

redis.call('SADD', participants_key, user_id)
local participant_count = redis.call('SCARD', participants_key)
local bid_count = redis.call('HINCRBY', state_key, 'bid_count', 1)

local bid = {
  id = bid_id,
  lot_id = lot_id,
  user_id = user_id,
  nickname = nickname,
  amount = amount,
  currency = currency,
  created_at_unix_ms = now_ms
}
local meta = {
  user_id = user_id,
  nickname = nickname,
  amount = amount,
  currency = currency,
  bid_at_unix_ms = now_ms,
  bid_id = bid_id
}
redis.call('ZADD', ranking_key, amount, user_id)
redis.call('HSET', rankmeta_key, user_id, cjson.encode(meta))
redis.call('LPUSH', recent_key, cjson.encode(bid))
redis.call('LTRIM', recent_key, 0, recent_limit - 1)

if status ~= status_settled and duel_active ~= 1 and bid_count >= 3 and ends_at - now_ms <= 60000 then
  local top = redis.call('ZREVRANGE', ranking_key, 0, 1, 'WITHSCORES')
  if #top >= 4 then
    local top_amount = tonumber(top[2] or '0')
    local second_amount = tonumber(top[4] or '0')
    if top_amount - second_amount <= min_increment_amount * 3 then
      local meta_a = cjson.decode(redis.call('HGET', rankmeta_key, top[1]) or '{}')
      local meta_b = cjson.decode(redis.call('HGET', rankmeta_key, top[3]) or '{}')
      duel_active = 1
      duel_user_a_id = top[1]
      duel_user_a_nickname = meta_a.nickname or ''
      duel_user_b_id = top[3]
      duel_user_b_nickname = meta_b.nickname or ''
      duel_started_at = now_ms
      duel_ends_at = ends_at
      playbook_stage = stage_duel
    end
  end
end

redis.call('HMSET', state_key,
  'status', status,
  'current_amount', amount,
  'current_currency', currency,
  'leading_user_id', user_id,
  'leading_nickname', nickname,
  'version', version,
  'playbook_stage', playbook_stage,
  'ends_at_unix_ms', ends_at,
  'settled_at_unix_ms', settled_at,
  'winner_user_id', winner_user_id,
  'winner_nickname', winner_nickname,
  'final_amount', final_amount,
  'final_currency', final_currency,
  'duel_active', duel_active,
  'duel_extend_count', extend_count,
  'duel_user_a_id', duel_user_a_id,
  'duel_user_a_nickname', duel_user_a_nickname,
  'duel_user_b_id', duel_user_b_id,
  'duel_user_b_nickname', duel_user_b_nickname,
  'duel_started_at_unix_ms', duel_started_at,
  'duel_ends_at_unix_ms', duel_ends_at,
  'participant_count', participant_count
)
redis.call('EXPIRE', state_key, 604800)
redis.call('EXPIRE', ranking_key, 604800)
redis.call('EXPIRE', rankmeta_key, 604800)
redis.call('EXPIRE', participants_key, 604800)
redis.call('EXPIRE', recent_key, 604800)

local lot = {
  lot_id = lot_id,
  room_id = room_id,
  status = status,
  current_amount = amount,
  current_currency = currency,
  leading_user_id = user_id,
  leading_nickname = nickname,
  started_at_unix_ms = tonumber(redis.call('HGET', state_key, 'started_at_unix_ms') or '0'),
  ends_at_unix_ms = ends_at,
  settled_at_unix_ms = settled_at,
  winner_user_id = winner_user_id,
  winner_nickname = winner_nickname,
  final_amount = final_amount,
  final_currency = final_currency,
  version = version,
  playbook_stage = playbook_stage,
  duel_active = duel_active,
  duel_extend_count = extend_count,
  duel_user_a_id = duel_user_a_id,
  duel_user_a_nickname = duel_user_a_nickname,
  duel_user_b_id = duel_user_b_id,
  duel_user_b_nickname = duel_user_b_nickname,
  duel_started_at_unix_ms = duel_started_at,
  duel_ends_at_unix_ms = duel_ends_at,
  duel_max_extend_count = tonumber(redis.call('HGET', state_key, 'max_extend_count') or '0'),
  bid_count = bid_count,
  participant_count = participant_count
}
local response = {
  ok = true,
  replayed = false,
  bid = bid,
  lot = lot,
  previous_leader_id = previous_leader_id,
  ends_before_bid = ends_before_bid,
  extend_count_before = extend_count_before
}
redis.call('SET', idem_key, cjson.encode(response), 'EX', idem_ttl)
return cjson.encode(response)
`)

type runtimeBidJSON struct {
	ID              string `json:"id"`
	LotID           string `json:"lot_id"`
	UserID          string `json:"user_id"`
	Nickname        string `json:"nickname"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	CreatedAtUnixMs int64  `json:"created_at_unix_ms"`
}

type runtimeRankMetaJSON struct {
	UserID      string `json:"user_id"`
	Nickname    string `json:"nickname"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	BidAtUnixMs int64  `json:"bid_at_unix_ms"`
	BidID       string `json:"bid_id"`
}

type runtimeLotJSON struct {
	LotID               string `json:"lot_id"`
	RoomID              string `json:"room_id"`
	Status              int32  `json:"status"`
	CurrentAmount       int64  `json:"current_amount"`
	CurrentCurrency     string `json:"current_currency"`
	LeadingUserID       string `json:"leading_user_id"`
	LeadingNickname     string `json:"leading_nickname"`
	StartedAtUnixMs     int64  `json:"started_at_unix_ms"`
	EndsAtUnixMs        int64  `json:"ends_at_unix_ms"`
	SettledAtUnixMs     int64  `json:"settled_at_unix_ms"`
	WinnerUserID        string `json:"winner_user_id"`
	WinnerNickname      string `json:"winner_nickname"`
	FinalAmount         int64  `json:"final_amount"`
	FinalCurrency       string `json:"final_currency"`
	Version             int64  `json:"version"`
	PlaybookStage       int32  `json:"playbook_stage"`
	DuelActive          int64  `json:"duel_active"`
	DuelExtendCount     int32  `json:"duel_extend_count"`
	DuelUserAID         string `json:"duel_user_a_id"`
	DuelUserANickname   string `json:"duel_user_a_nickname"`
	DuelUserBID         string `json:"duel_user_b_id"`
	DuelUserBNickname   string `json:"duel_user_b_nickname"`
	DuelStartedAtUnixMs int64  `json:"duel_started_at_unix_ms"`
	DuelEndsAtUnixMs    int64  `json:"duel_ends_at_unix_ms"`
	DuelMaxExtendCount  int32  `json:"duel_max_extend_count"`
	BidCount            int64  `json:"bid_count"`
	ParticipantCount    int64  `json:"participant_count"`
}

type runtimePlaceBidReply struct {
	OK                bool           `json:"ok"`
	Replayed          bool           `json:"replayed"`
	Message           string         `json:"message"`
	Bid               runtimeBidJSON `json:"bid"`
	Lot               runtimeLotJSON `json:"lot"`
	PreviousLeaderID  string         `json:"previous_leader_id"`
	EndsBeforeBid     int64          `json:"ends_before_bid"`
	ExtendCountBefore int32          `json:"extend_count_before"`
}

func (s *Store) HydrateLotRuntime(ctx context.Context, lot *v1.Lot) error {
	return s.syncLotRuntime(ctx, lot, false)
}

func (s *Store) SyncLotRuntime(ctx context.Context, lot *v1.Lot) error {
	return s.syncLotRuntime(ctx, lot, true)
}

func (s *Store) PlaceBidRuntime(ctx context.Context, lot *v1.Lot, req *v1.PlaceBidRequest, bidderID, nickname, bidID string, nowMs int64) (auction.RuntimeBidResult, error) {
	if lot == nil {
		return auction.RuntimeBidResult{}, fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if err := s.HydrateLotRuntime(ctx, lot); err != nil {
		return auction.RuntimeBidResult{}, err
	}
	keys := []string{
		runtimeStateKey(lot.Id),
		runtimeRankingKey(lot.Id),
		runtimeRankMetaKey(lot.Id),
		runtimeParticipantsKey(lot.Id),
		runtimeRecentKey(lot.Id),
		runtimeIdemKey(lot.Id, bidderID, req.GetIdempotencyKey()),
	}
	raw, err := runtimePlaceBidScript.Run(ctx, s.redis, keys,
		bidID,
		bidderID,
		nickname,
		strconv.FormatInt(req.GetAmount().GetAmount(), 10),
		req.GetAmount().GetCurrency(),
		strconv.FormatInt(nowMs, 10),
		strconv.FormatInt(int64(runtimeIdempotencyTTL/time.Second), 10),
		strconv.FormatInt(runtimeRecentLimit, 10),
		strconv.Itoa(int(v1.LotStatus_LOT_STATUS_LIVE)),
		strconv.Itoa(int(v1.LotStatus_LOT_STATUS_EXTENDED)),
		strconv.Itoa(int(v1.LotStatus_LOT_STATUS_SETTLED)),
		strconv.Itoa(int(v1.PlaybookStage_PLAYBOOK_STAGE_BIDDING_ACTIVE)),
		strconv.Itoa(int(v1.PlaybookStage_PLAYBOOK_STAGE_DUEL_MODE)),
		strconv.Itoa(int(v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY)),
	).Text()
	if err != nil {
		return auction.RuntimeBidResult{}, err
	}
	var reply runtimePlaceBidReply
	if err := json.Unmarshal([]byte(raw), &reply); err != nil {
		return auction.RuntimeBidResult{}, err
	}
	if !reply.OK {
		return auction.RuntimeBidResult{}, fmt.Errorf("%w: %s", apperr.ErrInvalidArgument, reply.Message)
	}
	bid := runtimeJSONToBid(reply.Bid)
	updatedLot := runtimeJSONToLot(lot, reply.Lot)
	if reply.Replayed {
		currentLot, err := s.loadRuntimeLot(ctx, lot)
		if err == nil {
			updatedLot = currentLot
		}
	}
	ranking, err := s.RankingRuntime(ctx, lot.Id, auction.RealtimeRankingLimit())
	if err != nil {
		return auction.RuntimeBidResult{}, err
	}
	recentBids, err := s.recentRuntimeBids(ctx, lot.Id, runtimeRecentLimit)
	if err != nil {
		return auction.RuntimeBidResult{}, err
	}
	return auction.RuntimeBidResult{
		Lot:               updatedLot,
		Bid:               bid,
		Ranking:           ranking,
		RecentBids:        recentBids,
		PreviousLeaderID:  reply.PreviousLeaderID,
		EndsBeforeBid:     reply.EndsBeforeBid,
		ExtendCountBefore: reply.ExtendCountBefore,
		Replayed:          reply.Replayed,
	}, nil
}

func (s *Store) SnapshotRuntime(ctx context.Context, current *v1.Lot) (*v1.RoomSnapshot, error) {
	if current == nil {
		return nil, fmt.Errorf("%w: current lot is required", apperr.ErrInvalidArgument)
	}
	if err := s.HydrateLotRuntime(ctx, current); err != nil {
		return nil, err
	}
	lot, err := s.loadRuntimeLot(ctx, current)
	if err != nil {
		return nil, err
	}
	ranking, err := s.RankingRuntime(ctx, current.Id, auction.RealtimeRankingLimit())
	if err != nil {
		return nil, err
	}
	recent, err := s.recentRuntimeBids(ctx, current.Id, runtimeRecentLimit)
	if err != nil {
		return nil, err
	}
	return &v1.RoomSnapshot{
		RoomId:           current.RoomId,
		CurrentLot:       lot,
		Ranking:          ranking,
		RecentBids:       recent,
		PlaybookStage:    lot.PlaybookStage,
		ServerTimeUnixMs: time.Now().UnixMilli(),
	}, nil
}

func (s *Store) RankingRuntime(ctx context.Context, lotID string, limit int64) ([]*v1.RankingItem, error) {
	if lotID == "" {
		return nil, fmt.Errorf("%w: lot id is required", apperr.ErrInvalidArgument)
	}
	stop := int64(-1)
	if limit > 0 {
		stop = limit - 1
	}
	rows, err := s.redis.ZRevRangeWithScores(ctx, runtimeRankingKey(lotID), 0, stop).Result()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []*v1.RankingItem{}, nil
	}
	userIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, fmt.Sprint(row.Member))
	}
	metaValues, err := s.redis.HMGet(ctx, runtimeRankMetaKey(lotID), userIDs...).Result()
	if err != nil {
		return nil, err
	}
	ranking := make([]*v1.RankingItem, 0, len(rows))
	for i, rawMeta := range metaValues {
		meta := runtimeRankMetaJSON{UserID: userIDs[i], Amount: int64(rows[i].Score)}
		if text, ok := rawMeta.(string); ok && text != "" {
			_ = json.Unmarshal([]byte(text), &meta)
		}
		ranking = append(ranking, &v1.RankingItem{
			Rank:        int32(i + 1),
			UserId:      meta.UserID,
			Nickname:    meta.Nickname,
			Amount:      &v1.Money{Amount: meta.Amount, Currency: meta.Currency},
			BidAtUnixMs: meta.BidAtUnixMs,
		})
	}
	return ranking, nil
}

func (s *Store) syncLotRuntime(ctx context.Context, lot *v1.Lot, overwrite bool) error {
	if lot == nil {
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	stateKey := runtimeStateKey(lot.Id)
	if !overwrite {
		exists, err := s.redis.Exists(ctx, stateKey).Result()
		if err != nil {
			return err
		}
		if exists > 0 {
			return nil
		}
		lockKey := runtimeLockKey(lot.Id)
		locked, err := s.redis.SetNX(ctx, lockKey, strconv.FormatInt(time.Now().UnixNano(), 10), 5*time.Second).Result()
		if err != nil {
			return err
		}
		if !locked {
			for i := 0; i < 20; i++ {
				time.Sleep(10 * time.Millisecond)
				exists, err = s.redis.Exists(ctx, stateKey).Result()
				if err != nil {
					return err
				}
				if exists > 0 {
					return nil
				}
			}
			return fmt.Errorf("%w: lot runtime hydrate is busy", apperr.ErrInvalidArgument)
		}
		defer s.redis.Del(ctx, lockKey)
		exists, err = s.redis.Exists(ctx, stateKey).Result()
		if err != nil {
			return err
		}
		if exists > 0 {
			return nil
		}
	}
	bids, err := s.ListByLot(ctx, lot.Id)
	if err != nil {
		return err
	}
	return s.writeLotRuntime(ctx, lot, bids)
}

func (s *Store) writeLotRuntime(ctx context.Context, lot *v1.Lot, bids []v1.Bid) error {
	stats := runtimeStatsFromBids(bids)
	lotForState := proto.Clone(lot).(*v1.Lot)
	lotForState.Stats = stats
	state := runtimeStateMap(lotForState)
	ranking := auction.BuildRanking(bids)

	pipe := s.redis.Pipeline()
	keys := []string{
		runtimeStateKey(lot.Id),
		runtimeRankingKey(lot.Id),
		runtimeRankMetaKey(lot.Id),
		runtimeParticipantsKey(lot.Id),
		runtimeRecentKey(lot.Id),
	}
	pipe.Del(ctx, keys...)
	pipe.HSet(ctx, runtimeStateKey(lot.Id), state)
	participants := make([]any, 0, stats.ParticipantCount)
	seen := make(map[string]bool)
	for _, bid := range bids {
		if bid.UserId != "" && !seen[bid.UserId] {
			seen[bid.UserId] = true
			participants = append(participants, bid.UserId)
		}
	}
	if len(participants) > 0 {
		pipe.SAdd(ctx, runtimeParticipantsKey(lot.Id), participants...)
	}
	for _, item := range ranking {
		if item.GetAmount() == nil {
			continue
		}
		meta := runtimeRankMetaJSON{
			UserID:      item.UserId,
			Nickname:    item.Nickname,
			Amount:      item.GetAmount().GetAmount(),
			Currency:    item.GetAmount().GetCurrency(),
			BidAtUnixMs: item.BidAtUnixMs,
		}
		payload, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		pipe.ZAdd(ctx, runtimeRankingKey(lot.Id), redis.Z{Score: float64(item.GetAmount().GetAmount()), Member: item.UserId})
		pipe.HSet(ctx, runtimeRankMetaKey(lot.Id), item.UserId, payload)
	}
	start := 0
	if int64(len(bids)) > runtimeRecentLimit {
		start = len(bids) - int(runtimeRecentLimit)
	}
	for i := start; i < len(bids); i++ {
		payload, err := json.Marshal(bidToRuntimeJSON(bids[i]))
		if err != nil {
			return err
		}
		pipe.LPush(ctx, runtimeRecentKey(lot.Id), payload)
	}
	for _, key := range keys {
		pipe.Expire(ctx, key, runtimeStateTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) loadRuntimeLot(ctx context.Context, base *v1.Lot) (*v1.Lot, error) {
	values, err := s.redis.HGetAll(ctx, runtimeStateKey(base.Id)).Result()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: lot runtime state is missing", apperr.ErrInvalidArgument)
	}
	return runtimeStateToLot(base, values), nil
}

func (s *Store) recentRuntimeBids(ctx context.Context, lotID string, limit int64) ([]*v1.Bid, error) {
	if limit <= 0 {
		limit = runtimeRecentLimit
	}
	rows, err := s.redis.LRange(ctx, runtimeRecentKey(lotID), 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	bids := make([]*v1.Bid, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		var payload runtimeBidJSON
		if err := json.Unmarshal([]byte(rows[i]), &payload); err != nil {
			return nil, err
		}
		bids = append(bids, runtimeJSONToBid(payload))
	}
	return bids, nil
}

func runtimeStateMap(lot *v1.Lot) map[string]any {
	rule := lot.GetRule()
	if rule == nil {
		rule = &v1.BidRule{}
	}
	current := lot.GetCurrentPrice()
	if current == nil {
		current = rule.GetStartPrice()
	}
	if current == nil {
		current = &v1.Money{}
	}
	final := lot.GetFinalPrice()
	if final == nil {
		final = &v1.Money{Currency: current.GetCurrency()}
	}
	minIncrement := rule.GetMinIncrement()
	if minIncrement == nil {
		minIncrement = &v1.Money{}
	}
	capAmount := int64(0)
	capCurrency := ""
	if rule.GetCapPrice() != nil {
		capAmount = rule.GetCapPrice().GetAmount()
		capCurrency = rule.GetCapPrice().GetCurrency()
	}
	duel := lot.GetDuelState()
	if duel == nil {
		duel = &v1.DuelState{}
	}
	stats := lot.GetStats()
	if stats == nil {
		stats = &v1.LotStats{}
	}
	return map[string]any{
		"lot_id":                    lot.Id,
		"room_id":                   lot.RoomId,
		"status":                    int32(lot.Status),
		"current_amount":            current.GetAmount(),
		"current_currency":          current.GetCurrency(),
		"leading_user_id":           lot.LeadingUserId,
		"leading_nickname":          lot.LeadingNickname,
		"started_at_unix_ms":        lot.StartedAtUnixMs,
		"ends_at_unix_ms":           lot.EndsAtUnixMs,
		"settled_at_unix_ms":        lot.SettledAtUnixMs,
		"winner_user_id":            lot.WinnerUserId,
		"winner_nickname":           lot.WinnerNickname,
		"final_amount":              final.GetAmount(),
		"final_currency":            final.GetCurrency(),
		"version":                   lot.Version,
		"playbook_stage":            int32(lot.PlaybookStage),
		"min_increment_amount":      minIncrement.GetAmount(),
		"min_increment_currency":    minIncrement.GetCurrency(),
		"cap_amount":                capAmount,
		"cap_currency":              capCurrency,
		"anti_snipe_window_seconds": rule.GetAntiSnipeWindowSeconds(),
		"anti_snipe_extend_seconds": rule.GetAntiSnipeExtendSeconds(),
		"max_extend_count":          rule.GetMaxExtendCount(),
		"duel_active":               boolInt(duel.GetActive()),
		"duel_extend_count":         duel.GetExtendCount(),
		"duel_user_a_id":            duel.GetUserAId(),
		"duel_user_a_nickname":      duel.GetUserANickname(),
		"duel_user_b_id":            duel.GetUserBId(),
		"duel_user_b_nickname":      duel.GetUserBNickname(),
		"duel_started_at_unix_ms":   duel.GetStartedAtUnixMs(),
		"duel_ends_at_unix_ms":      duel.GetEndsAtUnixMs(),
		"bid_count":                 stats.GetBidCount(),
		"participant_count":         stats.GetParticipantCount(),
	}
}

func runtimeStateToLot(base *v1.Lot, values map[string]string) *v1.Lot {
	lot := proto.Clone(base).(*v1.Lot)
	lot.Status = v1.LotStatus(parseInt32(values["status"]))
	lot.CurrentPrice = &v1.Money{Amount: parseInt64(values["current_amount"]), Currency: values["current_currency"]}
	lot.LeadingUserId = values["leading_user_id"]
	lot.LeadingNickname = values["leading_nickname"]
	lot.StartedAtUnixMs = parseInt64(values["started_at_unix_ms"])
	lot.EndsAtUnixMs = parseInt64(values["ends_at_unix_ms"])
	lot.SettledAtUnixMs = parseInt64(values["settled_at_unix_ms"])
	lot.WinnerUserId = values["winner_user_id"]
	lot.WinnerNickname = values["winner_nickname"]
	lot.FinalPrice = &v1.Money{Amount: parseInt64(values["final_amount"]), Currency: values["final_currency"]}
	lot.Version = parseInt64(values["version"])
	lot.PlaybookStage = v1.PlaybookStage(parseInt32(values["playbook_stage"]))
	lot.Stats = &v1.LotStats{
		ParticipantCount: parseInt64(values["participant_count"]),
		BidCount:         parseInt64(values["bid_count"]),
	}
	lot.DuelState = &v1.DuelState{
		Active:          parseInt64(values["duel_active"]) == 1,
		LotId:           lot.Id,
		UserAId:         values["duel_user_a_id"],
		UserANickname:   values["duel_user_a_nickname"],
		UserBId:         values["duel_user_b_id"],
		UserBNickname:   values["duel_user_b_nickname"],
		StartedAtUnixMs: parseInt64(values["duel_started_at_unix_ms"]),
		EndsAtUnixMs:    parseInt64(values["duel_ends_at_unix_ms"]),
		ExtendCount:     parseInt32(values["duel_extend_count"]),
		MaxExtendCount:  parseInt32(values["max_extend_count"]),
	}
	return lot
}

func runtimeJSONToLot(base *v1.Lot, payload runtimeLotJSON) *v1.Lot {
	lot := proto.Clone(base).(*v1.Lot)
	lot.Status = v1.LotStatus(payload.Status)
	lot.CurrentPrice = &v1.Money{Amount: payload.CurrentAmount, Currency: payload.CurrentCurrency}
	lot.LeadingUserId = payload.LeadingUserID
	lot.LeadingNickname = payload.LeadingNickname
	lot.StartedAtUnixMs = payload.StartedAtUnixMs
	lot.EndsAtUnixMs = payload.EndsAtUnixMs
	lot.SettledAtUnixMs = payload.SettledAtUnixMs
	lot.WinnerUserId = payload.WinnerUserID
	lot.WinnerNickname = payload.WinnerNickname
	lot.FinalPrice = &v1.Money{Amount: payload.FinalAmount, Currency: payload.FinalCurrency}
	lot.Version = payload.Version
	lot.PlaybookStage = v1.PlaybookStage(payload.PlaybookStage)
	lot.Stats = &v1.LotStats{ParticipantCount: payload.ParticipantCount, BidCount: payload.BidCount}
	lot.DuelState = &v1.DuelState{
		Active:          payload.DuelActive == 1,
		LotId:           lot.Id,
		UserAId:         payload.DuelUserAID,
		UserANickname:   payload.DuelUserANickname,
		UserBId:         payload.DuelUserBID,
		UserBNickname:   payload.DuelUserBNickname,
		StartedAtUnixMs: payload.DuelStartedAtUnixMs,
		EndsAtUnixMs:    payload.DuelEndsAtUnixMs,
		ExtendCount:     payload.DuelExtendCount,
		MaxExtendCount:  payload.DuelMaxExtendCount,
	}
	return lot
}

func runtimeStatsFromBids(bids []v1.Bid) *v1.LotStats {
	participants := make(map[string]bool)
	for _, bid := range bids {
		if bid.UserId != "" {
			participants[bid.UserId] = true
		}
	}
	return &v1.LotStats{ParticipantCount: int64(len(participants)), BidCount: int64(len(bids))}
}

func bidToRuntimeJSON(bid v1.Bid) runtimeBidJSON {
	return runtimeBidJSON{
		ID:              bid.Id,
		LotID:           bid.LotId,
		UserID:          bid.UserId,
		Nickname:        bid.Nickname,
		Amount:          bid.GetAmount().GetAmount(),
		Currency:        bid.GetAmount().GetCurrency(),
		CreatedAtUnixMs: bid.CreatedAtUnixMs,
	}
}

func runtimeJSONToBid(payload runtimeBidJSON) *v1.Bid {
	return &v1.Bid{
		Id:              payload.ID,
		LotId:           payload.LotID,
		UserId:          payload.UserID,
		Nickname:        payload.Nickname,
		Amount:          &v1.Money{Amount: payload.Amount, Currency: payload.Currency},
		CreatedAtUnixMs: payload.CreatedAtUnixMs,
	}
}

func runtimeTag(lotID string) string {
	return "auction:lot:{" + lotID + "}"
}

func runtimeStateKey(lotID string) string {
	return runtimeTag(lotID) + ":state"
}

func runtimeRankingKey(lotID string) string {
	return runtimeTag(lotID) + ":ranking"
}

func runtimeRankMetaKey(lotID string) string {
	return runtimeTag(lotID) + ":rankmeta"
}

func runtimeParticipantsKey(lotID string) string {
	return runtimeTag(lotID) + ":participants"
}

func runtimeRecentKey(lotID string) string {
	return runtimeTag(lotID) + ":recent"
}

func runtimeLockKey(lotID string) string {
	return runtimeTag(lotID) + ":hydrate_lock"
}

func runtimeIdemKey(lotID, userID, key string) string {
	return runtimeTag(lotID) + ":idem:" + userID + ":" + key
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func parseInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}

func parseInt32(value string) int32 {
	return int32(parseInt64(value))
}
