import http from 'node:http';
import { WebSocketServer } from 'ws';

const port = Number(process.env.MOCK_API_PORT || 8080);
const rooms = new Map();
let lot = {
  id: 'lot_demo_001',
  roomId: 'demo',
  title: '18K 金镶翡翠吊坠',
  description: '直播竞拍样品：支持实时出价、排行榜、AI 气氛官与落锤成交闭环。',
  imageUrl: 'https://images.unsplash.com/photo-1601121141461-9d6647bca1ed?w=900',
  startPrice: 188800,
  currentPrice: 188800,
  minIncrement: 5000,
  status: 'LIVE',
  endsAt: new Date(Date.now() + 30 * 60_000).toISOString(),
  winnerUserId: '',
  version: 1,
  bids: [],
  ranking: [],
  atmosphereText: '直播间已开拍，等待第一口出价。'
};

function rank() {
  const best = new Map();
  for (const b of lot.bids) {
    const old = best.get(b.userId);
    if (!old || b.amount > old.amount) best.set(b.userId, b);
  }
  lot.ranking = [...best.values()].sort((a,b)=>b.amount-a.amount || new Date(a.createdAt)-new Date(b.createdAt)).slice(0,10).map((b,i)=>({rank:i+1,userId:b.userId,nickname:b.nickname,amount:b.amount,at:b.createdAt}));
}
function sendJSON(res, data, code=200) {
  res.writeHead(code, {'Content-Type':'application/json','Access-Control-Allow-Origin':'*','Access-Control-Allow-Headers':'Content-Type','Access-Control-Allow-Methods':'GET,POST,OPTIONS'});
  res.end(JSON.stringify(data));
}
function broadcast(roomId, msg) {
  for (const ws of rooms.get(roomId) || []) if (ws.readyState === 1) ws.send(JSON.stringify(msg));
}
function applyBid({lotId,userId,nickname,amount}) {
  if (lotId !== lot.id) throw new Error('lot not found');
  const min = lot.currentPrice + lot.minIncrement;
  if (amount < min) throw new Error(`bid too low, minimum ${min}`);
  const bid = {id:`bid_${Date.now()}_${Math.random().toString(16).slice(2)}`, lotId, userId, nickname, amount, createdAt:new Date().toISOString()};
  lot.bids.push(bid);
  lot.currentPrice = amount;
  lot.winnerUserId = userId;
  lot.version += 1;
  lot.atmosphereText = `${nickname} 出价 ¥${(amount/100).toLocaleString()}，当前领先！离落锤又近一步。`;
  rank();
  broadcast(lot.roomId, {type:'bid.accepted', data:{bid, lot}});
  broadcast(lot.roomId, {type:'lot.updated', data:lot});
  return lot;
}

const server = http.createServer((req,res)=>{
  if (req.method === 'OPTIONS') return sendJSON(res, {});
  if (req.url === '/healthz') return sendJSON(res, {ok:true});
  if (req.url === '/api/lots' && req.method === 'GET') return sendJSON(res, [lot]);
  if (req.url === '/api/lots' && req.method === 'POST') {
    let body=''; req.on('data',c=>body+=c); req.on('end',()=>{
      const p = JSON.parse(body||'{}');
      lot = {...lot, ...p, id:`lot_${Date.now()}`, currentPrice:p.startPrice ?? lot.currentPrice, status:'LIVE', version:1, bids:[], ranking:[], endsAt:new Date(Date.now()+(p.durationSec||1200)*1000).toISOString()};
      sendJSON(res, lot); broadcast(lot.roomId, {type:'lot.updated', data:lot});
    }); return;
  }
  const m = req.url?.match(/^\/api\/lots\/([^/]+)\/bid$/);
  if (m && req.method === 'POST') {
    let body=''; req.on('data',c=>body+=c); req.on('end',()=>{
      try { sendJSON(res, applyBid({...JSON.parse(body||'{}'), lotId:m[1]})); }
      catch(e) { sendJSON(res, {error:e.message}, 409); }
    }); return;
  }
  sendJSON(res, {service:'live-auction-mock-api', ok:true});
});
const wss = new WebSocketServer({ server, path:'/ws/rooms/demo' });
wss.on('connection', ws => {
  const roomId='demo';
  if (!rooms.has(roomId)) rooms.set(roomId, new Set());
  rooms.get(roomId).add(ws);
  ws.send(JSON.stringify({type:'lot.updated', data:lot}));
  ws.on('message', raw => {
    try { const msg = JSON.parse(raw); if (msg.type === 'bid.place') applyBid(msg); }
    catch(e) { ws.send(JSON.stringify({type:'error', data:{message:e.message}})); }
  });
  ws.on('close',()=>rooms.get(roomId)?.delete(ws));
});
server.listen(port, '0.0.0.0', () => console.log(`mock api/ws listening on http://localhost:${port}`));
