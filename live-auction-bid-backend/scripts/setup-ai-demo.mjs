#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
loadLocalEnv(path.resolve(__dirname, '../deploy/.env'));

const baseUrl = (process.env.BASE_URL || process.env.AUCTION_API_BASE_URL || 'http://127.0.0.1:18080').replace(/\/+$/, '');
const pcUrl = (process.env.PC_URL || 'http://127.0.0.1:5173').replace(/\/+$/, '');
const h5Url = (process.env.H5_URL || 'http://127.0.0.1:5174').replace(/\/+$/, '');
const runId = safeRunId(process.env.RUN_ID || `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`);
const merchantUsername = process.env.DEMO_MERCHANT_USERNAME || `ai_demo_main_${runId}`;
const merchantPassword = process.env.DEMO_MERCHANT_PASSWORD || 'AiDemoPass123!';
const buyerPassword = process.env.DEMO_BUYER_PASSWORD || 'BuyerPass123!';
const buyerAUsername = process.env.DEMO_BUYER_A_USERNAME || `ai_demo_buyer_a_${runId}`;
const buyerBUsername = process.env.DEMO_BUYER_B_USERNAME || `ai_demo_buyer_b_${runId}`;
const demoDurationSeconds = parsePositiveInt(process.env.DEMO_DURATION_SECONDS, 12 * 60 * 60);

main().catch((error) => {
  console.error(error?.stack || error);
  process.exit(1);
});

async function main() {
  const merchant = await ensureMerchant(merchantUsername, merchantPassword, 'AI 演示商家');
  const merchantToken = tokenOf(merchant);
  const room = await firstAdminRoom(merchantToken);
  const image = await uploadDemoImage(merchantToken, room.id).catch((error) => {
    console.warn(`image upload failed, use public fallback image instead: ${safeMessage(error)}`);
    return {
      imageUrl: 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/live-anchor-01.jpg',
      fallback: true,
    };
  });

  const liveLot = await createDemoLot(merchantToken, room.id, {
    title: 'AI 演示拍品｜冰糯种翡翠手镯 56圈口',
    description: '用于 AI 竞拍咨询助手演示的公开拍品。冰糯种翡翠手镯，适合预算两万左右的用户咨询、检索和竞拍建议。',
    imageUrl: image.imageUrl,
    startPrice: 1800000,
    minIncrement: 20000,
    capPrice: 2600000,
    tags: ['翡翠手镯', '预算两万', 'AI演示', '收藏送礼'],
    category: '珠宝玉翠',
    estimatePrice: 2200000,
    stock: 1,
  });
  await queueLot(merchantToken, liveLot.id);
  const startedLot = await startLot(merchantToken, liveLot.id);

  const queuedLots = [];
  for (const spec of [
    {
      title: 'AI 演示待拍｜和田玉平安扣',
      description: '待拍演示拍品，供用户端 AI 搜索返回更多公开场次。',
      startPrice: 680000,
      minIncrement: 50000,
      capPrice: 1200000,
      tags: ['和田玉', '平安扣', '待拍'],
      category: '珠宝玉翠',
      estimatePrice: 980000,
      stock: 1,
    },
    {
      title: 'AI 演示待拍｜老银鎏金手作镯',
      description: '待拍演示拍品，帮助 AI 说明即将开始的拍品队列。',
      startPrice: 120000,
      minIncrement: 10000,
      capPrice: 360000,
      tags: ['银饰', '手作', '待拍'],
      category: '文玩配饰',
      estimatePrice: 260000,
      stock: 1,
    },
  ]) {
    const lot = await createDemoLot(merchantToken, room.id, { ...spec, imageUrl: image.imageUrl });
    await queueLot(merchantToken, lot.id);
    queuedLots.push(lot);
  }

  const buyerA = await ensureBuyer(buyerAUsername, buyerPassword, '演示买家A');
  const buyerB = await ensureBuyer(buyerBUsername, buyerPassword, '演示买家B');
  const bids = [
    [buyerA, 1820000],
    [buyerB, 1840000],
    [buyerA, 1860000],
    [buyerB, 1880000],
    [buyerA, 1900000],
  ];
  for (let index = 0; index < bids.length; index += 1) {
    const [buyer, amount] = bids[index];
    await placeBid(tokenOf(buyer), startedLot.id, amount, `ai-demo-${runId}-${index}`);
  }

  const snapshot = await getSnapshot(room.id);
  const currentLot = normalizeLot(snapshot.currentLot || snapshot.current_lot || startedLot);

  console.log(JSON.stringify({
    baseUrl,
    runId,
    merchant: {
      username: merchantUsername,
      password: merchantPassword,
    },
    buyers: [
      { username: buyerAUsername, password: buyerPassword },
      { username: buyerBUsername, password: buyerPassword },
    ],
    roomId: room.id,
    liveLotId: currentLot.id || startedLot.id,
    queuedLotIds: queuedLots.map((lot) => lot.id),
    durationSeconds: demoDurationSeconds,
    imageUploadedToTOS: !image.fallback,
    urls: {
      pcControl: `${pcUrl}/admin/auctions/current/control`,
      h5Room: `${h5Url}/m/room/${encodeURIComponent(room.id)}`,
      h5Search: `${h5Url}/home/search`,
    },
    recommendedSearch: '预算两万的翡翠手镯',
    expectedAIScene: 'Top2 接近、信任卡未揭示、可建议揭示信任卡或进入 Duel',
  }, null, 2));
}

async function ensureMerchant(username, password, nickname) {
  try {
    return await request('/api/merchants/register', { method: 'POST', body: { username, password, nickname } });
  } catch (error) {
    if (isNetworkFailure(error)) throw error;
    return request('/api/users/login', { method: 'POST', body: { username, password } });
  }
}

async function ensureBuyer(username, password, nickname) {
  try {
    return await request('/api/users/register', { method: 'POST', body: { username, password, nickname } });
  } catch (error) {
    if (isNetworkFailure(error)) throw error;
    return request('/api/users/login', { method: 'POST', body: { username, password } });
  }
}

async function firstAdminRoom(token) {
  const reply = await request('/api/admin/rooms', { token });
  const rooms = (reply.rooms || []).map(normalizeRoom);
  if (!rooms.length) throw new Error('admin room list is empty after merchant login');
  return rooms[0];
}

async function uploadDemoImage(token, roomId) {
  const form = new FormData();
  const bytes = Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAlgAAAIYCAIAAAB1nZcHAAAE+klEQVR4nO3VQQ0AIBDAsAP/nuGNAvZoFSzZnZk5BAAAfI4AAQKEAAFCgAAhQIAQIIC+3gHQe2fm3PUxAQBQgiBAgAAhQIAQIICAAAmgBAgQIAQI+AQIECAECBAgQAiQgAAhQIAQIICAAAkIQIAAIUCAECBAgAAJCECAl/l9AAAAcDwBAgQIAQKEAAFCgAAhQIAQIECAECBAgAAJECAECBBAgAAhQIAQIICAAAkIQIAAIUCAECBAgAAJCECAACFAgBAgQIAACSBAgAAhQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAACFAgBAgQIAQIECAECAACBAgQIAQIECAESEAAECBAgBAoQAAUKAAH29A8DKR+o8V70tAAAAAElFTkSuQmCC',
    'base64',
  );
  form.append('file', new Blob([bytes], { type: 'image/png' }), `ai-demo-${runId}.png`);
  form.append('roomId', roomId);
  form.append('bizType', 'lot_image');
  const reply = await request('/api/uploads/images', { method: 'POST', token, body: form });
  const asset = reply.data?.asset || reply.asset || {};
  const imageUrl = asset.imageUrl || asset.image_url;
  if (!imageUrl) throw new Error('upload reply missing imageUrl');
  return { imageUrl, assetId: asset.id || '' };
}

async function createDemoLot(token, roomId, spec) {
  const reply = await request('/api/lots', {
    method: 'POST',
    token,
    body: {
      room_id: roomId,
      title: spec.title,
      description: spec.description,
      image_url: spec.imageUrl,
      gallery_image_urls: [spec.imageUrl],
      category: spec.category,
      tags: spec.tags,
      estimate_price: money(spec.estimatePrice),
      stock: spec.stock,
      after_sale_notes: '平台担保交易，支持按页面规则售后咨询；竞拍前请确认尺寸、瑕疵和预算上限。',
      deposit_amount: money(0),
      trust_cards: trustCardsFor(spec.title),
      rule: {
        start_price: money(spec.startPrice),
        min_increment: money(spec.minIncrement),
        duration_seconds: demoDurationSeconds,
        anti_snipe_window_seconds: 20,
        anti_snipe_extend_seconds: 20,
        max_extend_count: 3,
        cap_price: money(spec.capPrice),
      },
    },
  });
  return normalizeLot(reply.lot);
}

async function queueLot(token, lotId) {
  const reply = await request(`/api/lots/${encodeURIComponent(lotId)}/queue`, {
    method: 'POST',
    token,
    body: { lotId },
  });
  return normalizeLot(reply.lot);
}

async function startLot(token, lotId) {
  const reply = await request(`/api/lots/${encodeURIComponent(lotId)}/start`, {
    method: 'POST',
    token,
    body: {},
  });
  return normalizeLot(reply.lot);
}

async function placeBid(token, lotId, amount, idempotencyKey) {
  const reply = await request(`/api/lots/${encodeURIComponent(lotId)}/bid`, {
    method: 'POST',
    token,
    idempotencyKey,
    body: {
      amount: money(amount),
      idempotency_key: idempotencyKey,
    },
    allowResultError: true,
  });
  if (!reply.accepted && !reply.bid?.id) {
    throw new Error(`bid rejected: ${reply.rejectReason || reply.reject_reason || reply.result?.message || 'unknown reason'}`);
  }
  return reply;
}

async function getSnapshot(roomId) {
  const reply = await request(`/api/rooms/${encodeURIComponent(roomId)}/snapshot`);
  return reply.snapshot || reply;
}

async function request(pathname, options = {}) {
  const headers = new Headers({ Accept: 'application/json' });
  const method = options.method || 'GET';
  let body;

  if (options.body instanceof FormData) {
    body = options.body;
  } else if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json');
    body = JSON.stringify(options.body);
  }

  if (options.token) headers.set('Authorization', `Bearer ${options.token}`);
  if (options.idempotencyKey) headers.set('Idempotency-Key', options.idempotencyKey);

  let response;
  try {
    response = await fetch(`${baseUrl}${pathname}`, { method, headers, body });
  } catch (error) {
    throw new Error(`${method} ${baseUrl}${pathname} network failed: ${safeMessage(error)}`);
  }

  const text = await response.text();
  let data = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      throw new Error(`${method} ${pathname} returned non-json ${response.status}: ${text.slice(0, 160)}`);
    }
  }
  if (!response.ok) throw new Error(`${method} ${pathname} HTTP ${response.status}: ${text.slice(0, 240)}`);

  const result = data.result;
  if (result && Number(result.code || 0) !== 0 && !options.allowResultError) {
    throw new Error(`${method} ${pathname} failed: ${result.message || result.code}`);
  }
  return data.data && !data.lot && !data.user && !data.tokens ? data.data : data;
}

function trustCardsFor(title) {
  return [
    {
      id: `trust-cert-${runId}`,
      type: 'TRUST_CARD_TYPE_CERTIFICATE',
      title: '证书与来源说明',
      content: `${title} 已准备证书/来源说明，建议主播在出价活跃前揭示，降低新用户疑虑。`,
    },
    {
      id: `trust-flaw-${runId}`,
      type: 'TRUST_CARD_TYPE_FLAW',
      title: '瑕疵与尺寸说明',
      content: '已标注圈口、尺寸和可见纹裂/使用痕迹，用户参拍前应确认是否接受。',
    },
    {
      id: `trust-service-${runId}`,
      type: 'TRUST_CARD_TYPE_SERVICE',
      title: '售后与复检说明',
      content: '平台担保交易，支持按页面规则售后咨询和复检说明，不建议私下交易。',
    },
    {
      id: `trust-price-${runId}`,
      type: 'TRUST_CARD_TYPE_PRICE_REF',
      title: '参考价说明',
      content: '参考价来自商家录入区间，仅作竞拍预算参考，最终成交价以实时竞拍结果为准。',
    },
  ];
}

function loadLocalEnv(envPath) {
  if (!fs.existsSync(envPath)) return;
  const content = fs.readFileSync(envPath, 'utf8');
  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#') || !line.includes('=')) continue;
    const index = line.indexOf('=');
    const key = line.slice(0, index).trim();
    const rawValue = line.slice(index + 1).trim();
    if (!key || Object.prototype.hasOwnProperty.call(process.env, key)) continue;
    process.env[key] = unquoteEnv(rawValue);
  }
}

function unquoteEnv(value) {
  if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
    return value.slice(1, -1);
  }
  return value;
}

function money(amount) {
  return { amount, currency: 'CNY' };
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

function safeRunId(value) {
  return String(value).replace(/[^a-zA-Z0-9_]/g, '_').slice(0, 40);
}

function parsePositiveInt(raw, fallback) {
  const value = Number.parseInt(String(raw || ''), 10);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

function safeMessage(error) {
  return error instanceof Error ? error.message : String(error);
}

function isNetworkFailure(error) {
  return safeMessage(error).includes('network failed');
}
