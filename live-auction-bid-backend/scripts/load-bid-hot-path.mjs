#!/usr/bin/env node

const baseUrl = (process.env.BASE_URL || 'http://127.0.0.1:18080').replace(/\/+$/, '');
const concurrency = parsePositiveInt(process.env.CONCURRENCY, 100);
const rankingLimit = parsePositiveInt(process.env.AUCTION_REALTIME_RANKING_LIMIT, 50);
const runId = process.env.RUN_ID || `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
const merchantUsername = process.env.MERCHANT_USERNAME || `load_main_${runId}`;
const merchantPassword = process.env.MERCHANT_PASSWORD || 'LoadTestPass123!';
const buyerPrefix = process.env.BUYER_PREFIX || `load_buyer_${runId}`;
const startPrice = parsePositiveInt(process.env.START_PRICE_CENTS, 10000);
const minIncrement = parsePositiveInt(process.env.MIN_INCREMENT_CENTS, 100);
const capPrice = startPrice + concurrency * minIncrement;

main().catch((error) => {
  console.error(error?.stack || error);
  process.exit(1);
});

async function main() {
  const merchant = await ensureMerchant();
  const merchantToken = tokenOf(merchant);
  const room = await firstAdminRoom(merchantToken);
  const lot = await createQueuedStartedLot(merchantToken, room.id);

  await assertPublicRoomVisible(room.id, 'queued/start setup');

  const buyers = await registerBuyers(concurrency);
  const bidTasks = buyers.map((buyer, index) => {
    const amount = startPrice + (index + 1) * minIncrement;
    const idempotencyKey = `load-bid-${runId}-${index}`;
    return timed(() => placeBid(buyer.token, lot.id, amount, idempotencyKey))
      .then((result) => ({ ...result, buyer, amount, idempotencyKey }))
      .catch((error) => ({ error, buyer, amount, idempotencyKey, ms: 0 }));
  });
  const bidResults = await Promise.all(bidTasks);
  const accepted = bidResults.filter((item) => !item.error && item.reply.accepted);
  const rejected = bidResults.filter((item) => !item.error && !item.reply.accepted);
  const errors = bidResults.filter((item) => item.error);

  if (accepted.length === 0) {
    throw new Error(`no accepted bids; rejected=${rejected.length} errors=${errors.length}`);
  }

  const highest = accepted.reduce((best, item) => acceptedBidAmount(item) > acceptedBidAmount(best) ? item : best, accepted[0]);
  const highestAmount = acceptedBidAmount(highest);
  const duplicate = await placeBid(highest.buyer.token, lot.id, highestAmount, highest.idempotencyKey);
  if (duplicate.accepted && highest.reply.bid?.id && duplicate.bid?.id && duplicate.bid.id !== highest.reply.bid.id) {
    throw new Error(`idempotency replay created a different bid: first=${highest.reply.bid.id} replay=${duplicate.bid.id}`);
  }

  const snapshot = await getSnapshot(room.id);
  const currentLot = snapshot.currentLot || snapshot.current_lot;
  const ranking = highest.reply.ranking?.length ? highest.reply.ranking : (snapshot.ranking || []);
  const winnerResult = await lotResult(highest.buyer.token, lot.id);
  const resultLot = winnerResult.lot || {};
  const finalPrice = moneyAmount(
    resultLot.finalPrice || resultLot.final_price ||
    currentLot?.finalPrice || currentLot?.final_price || currentLot?.currentPrice || currentLot?.current_price,
  );
  const leader = resultLot.winnerUserId || resultLot.winner_user_id || currentLot?.winnerUserId || currentLot?.winner_user_id || currentLot?.leadingUserId || currentLot?.leading_user_id;
  if (finalPrice !== highestAmount) {
    throw new Error(`final/current price mismatch: got=${finalPrice} want=${highestAmount}`);
  }
  if (leader !== highest.buyer.user.id) {
    throw new Error(`leader mismatch: got=${leader} want=${highest.buyer.user.id}`);
  }
  assertRanking(ranking, rankingLimit);

  let orderCount = 0;
  if (finalPrice >= capPrice) {
    if (winnerResult.order || winnerResult.orderId || winnerResult.order_id) orderCount = 1;
    if (orderCount !== 1) throw new Error(`cap settlement should create one visible winner order, got=${orderCount}`);
  }

  const latencies = bidResults.filter((item) => Number.isFinite(item.ms) && item.ms > 0).map((item) => item.ms).sort((a, b) => a - b);
  const report = {
    baseUrl,
    runId,
    roomId: room.id,
    lotId: lot.id,
    concurrency,
    total: bidResults.length,
    accepted: accepted.length,
    rejected: rejected.length,
    errors: errors.length,
    p50Ms: percentile(latencies, 50),
    p95Ms: percentile(latencies, 95),
    p99Ms: percentile(latencies, 99),
    finalPrice,
    leader,
    highestAcceptedBidder: highest.buyer.user.id,
    rankingLength: ranking.length,
    rankingLimit,
    orderCount,
  };
  console.log(JSON.stringify(report, null, 2));
}

async function ensureMerchant() {
  const body = { username: merchantUsername, password: merchantPassword, nickname: merchantUsername };
  try {
    return await request('/api/merchants/register', { method: 'POST', body });
  } catch (error) {
    if (String(error?.message || error).includes('network failed')) throw error;
    return request('/api/users/login', { method: 'POST', body: { username: merchantUsername, password: merchantPassword } });
  }
}

async function firstAdminRoom(token) {
  const reply = await request('/api/admin/rooms', { token });
  const rooms = reply.rooms || [];
  if (!rooms.length) throw new Error('admin room list is empty after merchant login');
  return normalizeRoom(rooms[0]);
}

async function createQueuedStartedLot(token, roomId) {
  const create = await request('/api/lots', {
    method: 'POST',
    token,
    body: {
      room_id: roomId,
      title: `Load hot path ${runId}`,
      description: 'repeatable concurrent bid hot-path smoke',
      image_url: 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/live-anchor-01.jpg',
      rule: {
        start_price: money(startPrice),
        min_increment: money(minIncrement),
        duration_seconds: 300,
        anti_snipe_window_seconds: 15,
        anti_snipe_extend_seconds: 15,
        max_extend_count: 3,
        cap_price: money(capPrice),
      },
    },
  });
  const lot = normalizeLot(create.lot);
  await request(`/api/lots/${encodeURIComponent(lot.id)}/queue`, { method: 'POST', token, body: {} });
  await assertPublicRoomVisible(roomId, 'queued lot');
  const started = await request(`/api/lots/${encodeURIComponent(lot.id)}/start`, { method: 'POST', token, body: {} });
  return normalizeLot(started.lot || lot);
}

async function assertPublicRoomVisible(roomId, stage) {
  const reply = await request('/api/rooms');
  const rooms = (reply.rooms || []).map(normalizeRoom);
  if (!rooms.some((room) => room.id === roomId)) {
    throw new Error(`public room ${roomId} is not visible after ${stage}`);
  }
}

async function registerBuyers(count) {
  const buyers = [];
  for (let index = 0; index < count; index += 1) {
    const username = `${buyerPrefix}_${index}`;
    const password = 'BuyerPass123!';
    let reply;
    try {
      reply = await request('/api/users/register', {
        method: 'POST',
        body: { username, password, nickname: `买家${index}` },
      });
    } catch (error) {
      if (String(error?.message || error).includes('network failed')) throw error;
      reply = await request('/api/users/login', { method: 'POST', body: { username, password } });
    }
    buyers.push({ user: normalizeUser(reply.user), token: tokenOf(reply), username });
  }
  return buyers;
}

async function placeBid(token, lotId, amount, idempotencyKey) {
  const reply = await request(`/api/lots/${encodeURIComponent(lotId)}/bid`, {
    method: 'POST',
    token,
    idempotencyKey,
    allowResultError: true,
    body: {
      amount: money(amount),
      idempotency_key: idempotencyKey,
    },
  });
  const bidAmount = moneyAmount(reply.bid?.amount);
  return {
    accepted: Boolean(reply.accepted || (reply.bid?.id && bidAmount > 0)),
    bid: reply.bid,
    ranking: reply.ranking || [],
    rejectReason: reply.rejectReason || reply.reject_reason || reply.result?.message || '',
  };
}

async function getSnapshot(roomId) {
  const reply = await request(`/api/rooms/${encodeURIComponent(roomId)}/snapshot`);
  return reply.snapshot || reply;
}

async function lotResult(token, lotId) {
  return request(`/api/lots/${encodeURIComponent(lotId)}/result`, { token });
}

async function timed(fn) {
  const started = performance.now();
  const reply = await fn();
  return { reply, ms: Math.round(performance.now() - started) };
}

async function request(path, options = {}) {
  const headers = new Headers({ Accept: 'application/json' });
  if (options.body !== undefined) headers.set('Content-Type', 'application/json');
  if (options.token) headers.set('Authorization', `Bearer ${options.token}`);
  if (options.idempotencyKey) headers.set('Idempotency-Key', options.idempotencyKey);
  const method = options.method || 'GET';
  let response;
  try {
    response = await fetch(`${baseUrl}${path}`, {
      method,
      headers,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    });
  } catch (error) {
    throw new Error(`${method} ${baseUrl}${path} network failed: ${error instanceof Error ? error.message : String(error)}`);
  }
  const text = await response.text();
  let data = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      throw new Error(`${method} ${path} returned non-json ${response.status}: ${text.slice(0, 160)}`);
    }
  }
  if (!response.ok) throw new Error(`${method} ${path} HTTP ${response.status}: ${text.slice(0, 240)}`);
  const result = data.result;
  if (result && Number(result.code || 0) !== 0 && !options.allowResultError) {
    throw new Error(`${method} ${path} failed: ${result.message || result.code}`);
  }
  return data;
}

function assertRanking(ranking, limit) {
  if (ranking.length > limit) throw new Error(`ranking length ${ranking.length} exceeds limit ${limit}`);
  for (let index = 1; index < ranking.length; index += 1) {
    const previous = moneyAmount(ranking[index - 1].amount);
    const current = moneyAmount(ranking[index].amount);
    if (previous < current) throw new Error(`ranking is not sorted at ${index}: ${previous} < ${current}`);
  }
}

function percentile(values, p) {
  if (!values.length) return 0;
  const index = Math.min(values.length - 1, Math.ceil((p / 100) * values.length) - 1);
  return values[index];
}

function parsePositiveInt(raw, fallback) {
  const value = Number.parseInt(String(raw || ''), 10);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

function money(amount) {
  return { amount, currency: 'CNY' };
}

function moneyAmount(value) {
  if (!value) return 0;
  return Number(value.amount || 0);
}

function acceptedBidAmount(item) {
  return moneyAmount(item?.reply?.bid?.amount) || item?.amount || 0;
}

function tokenOf(reply) {
  const tokens = reply.tokens || {};
  const token = tokens.accessToken || tokens.access_token;
  if (!token) throw new Error('missing access token');
  return token;
}

function normalizeRoom(room) {
  return { ...room, id: room?.id || room?.room_id || '' };
}

function normalizeLot(lot) {
  return { ...lot, id: lot?.id || lot?.lot_id || '', roomId: lot?.roomId || lot?.room_id || '' };
}

function normalizeUser(user) {
  return { ...user, id: user?.id || user?.user_id || '' };
}
