import { type CSSProperties, useState } from 'react';
import { LivePlayer } from '../../live/components/LivePlayer';
import { DepositPayModal } from '../../payment-flow/components/DepositPayModal';
import { MockPayModal } from '../../payment-flow/components/MockPayModal';
import { ResultModal } from '../../result-modal/components/ResultModal';
import { businessErrorMessage } from '../../../shared/api/errors';
import { formatMoney, moneyNumber } from '../../../shared/lib/money';
import { navigateTo } from '../../../shared/navigation';
import type { LiveRoomController } from '../hooks/useLiveRoomController';
import { AuctionDrawer } from './AuctionDrawer';
import { AuctionNoticeLayer } from './AuctionNoticeLayer';

function shortCount(value: number): string {
  if (value >= 10000) return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}w`;
  return value.toLocaleString('zh-CN');
}

const SHARE_OPTIONS = ['转发', '私信朋友', '复制链接', '生成图片', '举报', '不感兴趣'];
const SHARE_FRIENDS = ['拍友A', '严选群', '收藏顾问', '直播搭子'];
const GIFT_OPTIONS = [
  { name: '小心心', value: '1', image: '/douyin-assets/gifts/xiao-xin-xin.png' },
  { name: '棒棒糖', value: '9', image: '/douyin-assets/gifts/bang-bang-tang.png' },
  { name: '人气票', value: '18', image: '/douyin-assets/gifts/ren-qi-piao.png' },
  { name: '烟花', value: '66', image: '/douyin-assets/gifts/yan-hua.png' },
  { name: '星河', value: '188', image: '/douyin-assets/gifts/xing-he.png' },
  { name: '嘉年华', value: '999', image: '/douyin-assets/gifts/jia-nian-hua.png' },
];
const MORE_ACTIONS = ['清屏', '小窗播放', '画质', '直播公告', '订单', '举报'];
const LIVE_AVATAR_POOL = [
  'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-71158770-d8597.jpeg',
  'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-lsy0508-160edjy.jpeg',
  'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-ll991221-1bmdvg4.jpeg',
  'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-sunmeng333-qheb8m.jpeg',
  'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-jingyiziran-176539n.jpeg',
  'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-8357999-1bd1vnm.jpeg',
];

function stableAvatarIndex(key: string): number {
  return Array.from(key || 'live-room').reduce((hash, char) => (hash * 31 + char.charCodeAt(0)) % LIVE_AVATAR_POOL.length, 0);
}

function liveAvatarFor(key: string, offset = 0): string {
  return LIVE_AVATAR_POOL[(stableAvatarIndex(key) + offset) % LIVE_AVATAR_POOL.length];
}

function firstNameChar(name: string): string {
  return name.trim().slice(0, 1) || '拍';
}

function AvatarMedia({ src, name }: { src: string; name: string }) {
  const [failed, setFailed] = useState(false);
  if (failed) return <>{firstNameChar(name)}</>;
  return <img src={src} alt="" referrerPolicy="no-referrer" loading="lazy" onError={() => setFailed(true)} />;
}

function LiveRoomChrome({ controller }: { controller: LiveRoomController }) {
  const { anchorName, currentLot, room } = controller;
  const likes = Math.max(room.snapshot?.onlineCount || 0, 93000);
  const viewers = room.ranking.slice(0, 3);
  const viewerAvatars = viewers.length ? viewers.map((viewer, index) => {
    const name = viewer.nickname || viewer.userId || `拍友${index + 1}`;
    return {
      id: viewer.userId || `viewer-${index}`,
      name,
      avatarUrl: liveAvatarFor(`${room.roomId}:${name}`, index + 1),
    };
  }) : [
    { id: 'fallback-buyer', name: '拍友', avatarUrl: liveAvatarFor(`${room.roomId}:buyer`, 1) },
    { id: 'fallback-collector', name: '藏家', avatarUrl: liveAvatarFor(`${room.roomId}:collector`, 2) },
    { id: 'fallback-bidder', name: '竞拍', avatarUrl: liveAvatarFor(`${room.roomId}:bidder`, 3) },
  ];
  const topRank = Math.max(1, Math.min(99, Number(String(room.roomId).replace(/\D/g, '').slice(-2)) || 15));
  const closeRoom = () => {
    if (window.history.length > 1) {
      window.history.back();
      return;
    }

    navigateTo('/home', { replace: true });
  };

  return (
    <section className="liveRoomChrome" aria-label="直播间信息">
      <div className="liveRoomTop">
        <div className="liveRoomTopLeft">
          <div className="liveAnchorLine">
            <div className="anchorAvatar">
              <AvatarMedia src={liveAvatarFor(`${room.roomId}:${anchorName}:anchor`)} name={anchorName} />
            </div>
            <div className="anchorCopy">
              <strong>{anchorName}</strong>
              <span>{shortCount(likes)}本场点赞</span>
            </div>
            <button type="button" className="followAnchor" onClick={() => controller.actions.showNotice('已关注主播')}>关注</button>
          </div>
          <div className="liveAnchorTags" aria-label="直播标签">
            <span>讲解</span>
            <span>严选第{topRank}名</span>
            <button type="button" onClick={() => controller.actions.showNotice(currentLot ? `正在竞拍 ${currentLot.title}` : '等待主播上架商品')}>
              {currentLot ? '竞拍中' : '等待上架'} ›
            </button>
          </div>
        </div>
        <div className="liveRoomTopRight">
          <div className="liveViewerStack" aria-label="在线观众">
            {viewerAvatars.map((viewer, index) => (
              <span key={`${viewer.id}-${index}`}>
                <AvatarMedia src={viewer.avatarUrl} name={viewer.name} />
              </span>
            ))}
            <b>{shortCount(Math.max(room.snapshot?.onlineCount || 0, 107))}</b>
            <button type="button" className="closeLive" aria-label="关闭直播间" onClick={closeRoom}>×</button>
          </div>
          <button type="button" className="moreCity" onClick={() => controller.actions.showNotice('更多同城直播暂未开放')}>
            <span>更多同城</span>
            <i>›</i>
          </button>
        </div>
      </div>
    </section>
  );
}

type LeaderboardEntry = LiveRoomController['ranking'][number];

function uniqueLeaderboardEntries(entries: LeaderboardEntry[]): LeaderboardEntry[] {
  const byBidder = new Map<string, LeaderboardEntry>();
  entries.forEach((entry, index) => {
    const key = entry.isMe ? 'me' : entry.userId || entry.nickname || `rank-${index}`;
    const existing = byBidder.get(key);
    if (!existing || moneyNumber(entry.amount) > moneyNumber(existing.amount)) byBidder.set(key, entry);
  });

  return Array.from(byBidder.values())
    .sort((a, b) => moneyNumber(b.amount) - moneyNumber(a.amount))
    .map((entry, index) => ({ ...entry, rank: index + 1 }));
}

function LiveBidLeaderboard({
  controller,
  onOpenAuction,
  collapsed,
  onToggleCollapsed,
}: {
  controller: LiveRoomController;
  onOpenAuction: () => void;
  collapsed: boolean;
  onToggleCollapsed: () => void;
}) {
  const { currentLot, ranking, room } = controller;
  if (!currentLot) return null;

  const acceptedRecentBids = room.recentBids.filter((bid) => bid.accepted !== false && (!bid.lotId || bid.lotId === currentLot.id));
  const bidCount = Math.max(currentLot.stats.bidCount, acceptedRecentBids.length);
  const uniqueRanking = uniqueLeaderboardEntries(ranking);
  const participantCount = Math.max(currentLot.stats.participantCount, uniqueRanking.length);
  const fallbackLeader: LeaderboardEntry | null = !uniqueRanking.length && (currentLot.leadingNickname || currentLot.leadingUserId)
    ? {
        rank: 1,
        userId: currentLot.leadingUserId || 'leader',
        nickname: currentLot.leadingNickname || '领先拍友',
        amount: currentLot.currentPrice,
        isMe: currentLot.leadingUserId === controller.meId,
      }
    : null;
  const entries = (uniqueRanking.length ? uniqueRanking : fallbackLeader ? [fallbackLeader] : []).slice(0, 3);
  const hasBid = bidCount > 0 || entries.length > 0;
  const leadAmountText = hasBid ? formatMoney(entries[0]?.amount || currentLot.currentPrice) : '未出价';

  if (collapsed) {
    return (
      <button
        type="button"
        className={`liveBidLeaderboard isCollapsed${hasBid ? '' : ' empty'}`}
        onClick={onToggleCollapsed}
        aria-label="展开竞拍排行榜"
      >
        <span className="leaderboardCollapsedPill">
          <span>{hasBid ? '竞拍榜' : '等首拍'}</span>
          <strong title={leadAmountText}>{leadAmountText}</strong>
          <i aria-hidden="true">›</i>
        </span>
      </button>
    );
  }

  return (
    <section
      className={`liveBidLeaderboard${hasBid ? '' : ' empty'}`}
      aria-label="竞拍排行榜"
    >
      <button type="button" className="leaderboardTop" onClick={onToggleCollapsed} aria-label="收起竞拍排行榜">
        <span className="leaderboardTitle">
          <b>{hasBid ? '竞拍榜' : '等首拍'}</b>
          <small>{participantCount}人围观 · {bidCount}口</small>
        </span>
        <strong title={leadAmountText}>{leadAmountText}</strong>
        <i className="leaderboardChevron" aria-hidden="true" />
      </button>
      <button type="button" className="leaderboardRows" onClick={onOpenAuction} aria-label="打开商品橱窗查看出价">
        {entries.length ? entries.map((entry, index) => {
          const name = entry.nickname || (entry.userId ? `拍友${entry.userId.slice(-4)}` : `拍友${index + 1}`);
          const amountText = formatMoney(entry.amount);
          return (
            <span className={`leaderboardRow${entry.isMe ? ' me' : ''}`} key={`${entry.userId || name}-${index}`}>
              <em>{entry.rank || index + 1}</em>
              <span className="leaderboardAvatar">
                <AvatarMedia src={entry.avatarUrl || liveAvatarFor(`${room.roomId}:${name}:rank`, index)} name={name} />
              </span>
              <b>{entry.isMe ? '我' : name}</b>
              <strong title={amountText}>{amountText}</strong>
            </span>
          );
        }) : <span className="leaderboardEmpty">等第一口出价</span>}
      </button>
    </section>
  );
}

type GiftBurst = { id: number; name: string };

function LiveRoomEffectsLayer({ controller, giftBurst }: { controller: LiveRoomController; giftBurst: GiftBurst | null }) {
  const { anchorName, currentLot, room, notices } = controller;
  type ChatMessage = { id: string; name: string; text: string; system?: boolean };
  const seededMessages = room.recentBids.slice(0, 4).map((bid, index) => ({
    id: bid.id || `${bid.userId}-${index}`,
    name: bid.nickname || `拍友${bid.userId.slice(-4)}`,
    text: bid.accepted === false
      ? businessErrorMessage(bid.rejectReason, { lot: currentLot || undefined }) || '刚刚互动未成功'
      : `刚刚关注了这件商品`,
  }));
  const messages: ChatMessage[] = seededMessages.length ? seededMessages : [
    { id: 'demo-welcome', name: anchorName, text: currentLot ? `正在竞拍 ${currentLot.title}` : '欢迎来到直播间' },
    { id: 'demo-detail', name: '拍友7281', text: '主播可以再看一下细节吗' },
    { id: 'demo-product', name: '收藏新手', text: '等主播讲到细节我再看商品' },
  ];
  const safetyNotice = {
    id: 'system-safe-notice',
    name: '系统提示',
    text: '平台倡导理性消费，请勿私下交易，订单和支付以 LiveAuction 页面为准。',
    system: true,
  };
  const chatMessages: ChatMessage[] = [safetyNotice, ...messages];
  const joiners = (room.ranking.length ? room.ranking : [
    { userId: 'demo-a', nickname: '严选拍友' },
    { userId: 'demo-b', nickname: '收藏顾问' },
    { userId: 'demo-c', nickname: '拍场观察' },
  ]).slice(0, 3);
  const barrage = [
    currentLot ? `${currentLot.title} 正在竞拍` : `${anchorName} 正在直播`,
    notices[0] || '喜欢这件可以打开商品',
    room.recentBids[0] ? `${room.recentBids[0].nickname || '拍友'} 刚刚关注商品` : '直播间热度上升',
  ];

  return (
    <section className="liveEffectsLayer" aria-label="直播互动动态">
      <div className="liveBarrageLane" aria-hidden="true">
        {barrage.map((text, index) => <span key={`${text}-${index}`}>{text}</span>)}
      </div>
      <div className="liveJoinTicker" aria-hidden="true">
        {joiners.map((viewer, index) => (
          <span key={`${viewer.userId}-${index}`}>{viewer.nickname || viewer.userId} 进入直播间</span>
        ))}
      </div>
      <div className="liveGiftBurst" aria-hidden="true">
        <i />
        <span>{currentLot ? '商品热度上升' : '直播间热度上升'}</span>
      </div>
      {giftBurst ? (
        <div className="liveGiftCombo" key={giftBurst.id} aria-hidden="true">
          <span className="giftSender">{anchorName.slice(0, 1)}</span>
          <p><b>{anchorName}</b><small>送出 {giftBurst.name}</small></p>
          <i>{giftBurst.name.slice(0, 1)}</i>
          <strong>x1</strong>
        </div>
      ) : null}
      <div className="liveChatStream">
        {chatMessages.map((message, index) => (
          <p className={message.system ? 'system' : ''} key={message.id} style={{ '--live-chat-index': index } as CSSProperties}>
            <b>{message.name}</b>
            <span>{message.text}</span>
          </p>
        ))}
      </div>
    </section>
  );
}

function ComposerIcon({ name }: { name: 'product' | 'gift' | 'more' | 'heart' }) {
  if (name === 'product') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M6.4 8.4h11.2l1.1 10.4a2 2 0 0 1-2 2.2H7.3a2 2 0 0 1-2-2.2L6.4 8.4Z" />
        <path d="M9 8.4V6.7a3 3 0 0 1 6 0v1.7" />
      </svg>
    );
  }
  if (name === 'gift') {
    return <img className="composerImageIcon composerGiftIcon" src="/douyin-assets/gifts/gift-icon.png" alt="" aria-hidden="true" loading="lazy" />;
  }
  if (name === 'more') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M5 12h.01M12 12h.01M19 12h.01" />
      </svg>
    );
  }
  return <img className="composerImageIcon composerHeartIcon" src="/douyin-assets/gifts/xiao-xin-xin.png" alt="" aria-hidden="true" loading="lazy" />;
}

function LiveComposer({
  controller,
  onOpenComments,
  onOpenAuction,
  onOpenGift,
  onOpenMore,
}: {
  controller: LiveRoomController;
  onOpenComments: () => void;
  onOpenAuction: () => void;
  onOpenGift: () => void;
  onOpenMore: () => void;
}) {
  const sendFacadeNotice = (message: string) => controller.actions.showNotice(message);

  return (
    <footer className="liveComposer" aria-label="直播互动输入">
      <button type="button" className="commentInput" onClick={onOpenComments}>说点什么...</button>
      <button type="button" className="composerIconButton" aria-label="商品橱窗" onClick={onOpenAuction}>
        <ComposerIcon name="product" />
      </button>
      <button type="button" className="composerIconButton" aria-label="礼物" onClick={onOpenGift}>
        <ComposerIcon name="gift" />
      </button>
      <button type="button" className="composerIconButton" aria-label="更多操作" onClick={onOpenMore}>
        <ComposerIcon name="more" />
      </button>
      <button type="button" className="composerIconButton" aria-label="点赞直播" onClick={() => sendFacadeNotice('已为直播间增加热度')}>
        <ComposerIcon name="heart" />
      </button>
    </footer>
  );
}

function commentsForRoom(controller: LiveRoomController) {
  const bidComments = controller.room.recentBids.slice(0, 8).map((bid, index) => ({
    id: bid.id || `${bid.userId}-${index}`,
    avatar: (bid.nickname || bid.userId || '拍').slice(0, 1),
    name: bid.nickname || `拍友${bid.userId.slice(-4)}`,
    text: bid.accepted === false
      ? businessErrorMessage(bid.rejectReason, { lot: controller.currentLot || undefined }) || '刚刚互动未成功，准备再试一次'
      : `关注了这件商品，挺有意思`,
    meta: index < 2 ? '刚刚' : `${index + 1}分钟前`,
    count: Math.max(3, 168 - index * 17),
  }));

  if (bidComments.length) return bidComments;

  return [
    {
      id: 'anchor-welcome',
      avatar: controller.anchorName.slice(0, 1),
      name: controller.anchorName,
      text: controller.currentLot ? `正在看 ${controller.currentLot.title}，喜欢可以点右侧商品` : '欢迎来到直播间，商品上架后会在底部出现',
      meta: '置顶',
      count: 256,
    },
    { id: 'buyer-1', avatar: '拍', name: '拍友7281', text: '这场节奏可以，等开拍', meta: '1分钟前', count: 88 },
    { id: 'buyer-2', avatar: '藏', name: '收藏新手', text: '能看到商品细节图吗', meta: '2分钟前', count: 42 },
  ];
}

function DouyinCommentsSheet({ controller, onClose }: { controller: LiveRoomController; onClose: () => void }) {
  const [likedIds, setLikedIds] = useState<Set<string>>(new Set());
  const comments = commentsForRoom(controller);
  const totalCount = comments.length + Math.max(controller.room.recentBids.length, 128);
  const toggleLike = (id: string) => {
    setLikedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  return (
    <div className="dyCommentOverlay" onClick={onClose}>
      <section className="dyCommentSheet" role="dialog" aria-modal="true" aria-label="直播评论" onClick={(e) => e.stopPropagation()}>
        <header className="dyCommentHeader">
          <div className="dyCommentHeaderLeft">
            <button type="button" className="dyCommentCloseBtn" onClick={onClose} aria-label="关闭评论">‹</button>
            <span className="dyCommentTitle">{shortCount(totalCount)}条评论</span>
          </div>
          <div className="dyCommentHeaderRight">
            <button type="button" aria-label="全屏">⤢</button>
            <button type="button" onClick={onClose} aria-label="关闭">✕</button>
          </div>
        </header>

        <div className="dyCommentList">
          {comments.map((comment) => (
            <div className="dyCommentItem" key={comment.id}>
              <div className="dyCommentAvatar">
                <span className="dyCommentAvatarPlaceholder">{comment.avatar}</span>
              </div>
              <div className="dyCommentBody">
                <div className="dyCommentName">{comment.name}</div>
                <p className="dyCommentText">{comment.text}</p>
                <div className="dyCommentMeta">
                  <span className="dyCommentTime">{comment.meta}</span>
                  <button type="button" className="dyCommentReply">回复</button>
                </div>
              </div>
              <div className="dyCommentActions">
                <button
                  type="button"
                  className={`dyCommentLike${likedIds.has(comment.id) ? ' active' : ''}`}
                  onClick={() => toggleLike(comment.id)}
                  aria-label="点赞评论"
                >
                  <svg viewBox="0 0 48 48"><path fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M15 8C8.925 8 4 12.925 4 19c0 11 13 21 20 23.326C31 40 44 30 44 19c0-6.075-4.925-11-11-11c-3.72 0-7.01 1.847-9 4.674A10.99 10.99 0 0 0 15 8" /></svg>
                  <span>{shortCount(comment.count + (likedIds.has(comment.id) ? 1 : 0))}</span>
                </button>
              </div>
            </div>
          ))}
          <div className="dyCommentNoMore">暂时没有更多了</div>
        </div>

        <div className="dyCommentToolbar">
          <input className="dyCommentInput" placeholder="善语结善缘，恶言伤人心" readOnly />
          <button type="button" className="dyCommentToolBtn" aria-label="表情">😊</button>
          <button type="button" className="dyCommentToolBtn" aria-label="@提及">@</button>
        </div>
      </section>
    </div>
  );
}

function DouyinShareSheet({ controller, onClose }: { controller: LiveRoomController; onClose: () => void }) {
  const copyShareLink = () => {
    void navigator.clipboard?.writeText(window.location.href).catch(() => undefined);
    controller.actions.showNotice('直播间链接已复制');
    onClose();
  };

  return (
    <div className="douyinSheetMask dark" onClick={onClose}>
      <section className="douyinBottomSheet shareSheet" role="dialog" aria-modal="true" aria-label="分享直播间" onClick={(event) => event.stopPropagation()}>
        <header className="shareHeader">
          <b>分享给朋友</b>
          <button type="button" className="sheetClose" onClick={onClose} aria-label="关闭分享">×</button>
        </header>
        <div className="shareFriendRow">
          {SHARE_FRIENDS.map((friend) => (
            <button type="button" className="shareFriend" key={friend}>
              <span>{friend.slice(0, 1)}</span>
              <b>{friend}</b>
            </button>
          ))}
          <button type="button" className="shareFriend moreFriend">
            <span>›</span>
            <b>更多朋友</b>
          </button>
        </div>
        <div className="shareOptionRow">
          {SHARE_OPTIONS.map((option) => (
            <button type="button" className="shareOption" key={option} onClick={option === '复制链接' ? copyShareLink : undefined}>
              <span>{option.slice(0, 1)}</span>
              <b>{option}</b>
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}

function DouyinGiftSheet({
  controller,
  onClose,
  onSendGift,
}: {
  controller: LiveRoomController;
  onClose: () => void;
  onSendGift: (name: string) => void;
}) {
  const sendGift = (name: string) => {
    onSendGift(name);
    onClose();
  };

  return (
    <div className="douyinSheetMask dark" onClick={onClose}>
      <section className="douyinBottomSheet giftSheet" role="dialog" aria-modal="true" aria-label="礼物" onClick={(event) => event.stopPropagation()}>
        <header className="shareHeader">
          <b>礼物</b>
          <button type="button" className="sheetClose" onClick={onClose} aria-label="关闭礼物">×</button>
        </header>
        <div className="giftBalance">
          <span>抖币余额 0</span>
          <button type="button" onClick={() => controller.actions.showNotice('充值功能暂未开放')}>充值</button>
        </div>
        <div className="giftGrid">
          {GIFT_OPTIONS.map((gift) => (
            <button type="button" onClick={() => sendGift(gift.name)} key={gift.name}>
              <span><img src={gift.image} alt="" loading="lazy" /></span>
              <b>{gift.name}</b>
              <small>{gift.value} 抖币</small>
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}

function DouyinMoreSheet({
  controller,
  onClose,
  onOpenAuction,
  onOpenShare,
  onToggleClear,
}: {
  controller: LiveRoomController;
  onClose: () => void;
  onOpenAuction: () => void;
  onOpenShare: () => void;
  onToggleClear: () => void;
}) {
  const runAction = (action: string) => {
    if (action === '清屏') {
      onClose();
      onToggleClear();
      return;
    }
    if (action === '订单') {
      onClose();
      window.location.assign(`/m/history?roomId=${encodeURIComponent(controller.roomId)}&from=live-more`);
      return;
    }
    if (action === '直播公告') {
      controller.actions.showNotice(controller.currentLot ? `正在竞拍 ${controller.currentLot.title}` : '欢迎来到直播间');
      onClose();
      return;
    }
    controller.actions.showNotice(`${action}暂未开放`);
    onClose();
  };

  return (
    <div className="douyinSheetMask dark" onClick={onClose}>
      <section className="douyinBottomSheet moreSheet" role="dialog" aria-modal="true" aria-label="更多操作" onClick={(event) => event.stopPropagation()}>
        <header className="shareHeader">
          <b>更多</b>
          <button type="button" className="sheetClose" onClick={onClose} aria-label="关闭更多">×</button>
        </header>
        <div className="moreActionGrid">
          <button type="button" onClick={() => { onClose(); onOpenAuction(); }}>
            <span>商</span>
            <b>商品橱窗</b>
          </button>
          <button type="button" onClick={() => { onClose(); onOpenShare(); }}>
            <span>享</span>
            <b>分享直播</b>
          </button>
          {MORE_ACTIONS.map((action) => (
            <button type="button" onClick={() => runAction(action)} key={action}>
              <span>{action.slice(0, 1)}</span>
              <b>{action}</b>
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}

export function LiveRoomView({ controller }: { controller: LiveRoomController }) {
  const [activeSheet, setActiveSheet] = useState<'comments' | 'share' | 'gift' | 'more' | null>(null);
  const [clearScreen, setClearScreen] = useState(false);
  const [giftBurst, setGiftBurst] = useState<GiftBurst | null>(null);
  const [leaderboardCollapsed, setLeaderboardCollapsed] = useState(false);
  const {
    room,
    error,
    roomName,
    anchorName,
    currentLot,
    meId,
    wsState,
    notices,
    auctionPanel,
    resultLot,
    visibleResultOrder,
    payOrder,
    depositPrompt,
    actions,
  } = controller;
  const openCurrentAuction = () => actions.openAuctionPanel('current');
  const closeActiveSheet = () => setActiveSheet(null);
  const sendGift = (name: string) => {
    setGiftBurst({ id: Date.now(), name });
    actions.showNotice(`已送出 ${name}`);
    window.setTimeout(() => setGiftBurst(null), 2300);
  };

  return (
    <main className={`mobileShell douyinShell ${auctionPanel.open ? 'drawerVisible' : ''} ${clearScreen ? 'isClearScreen' : ''}`}>
      <LivePlayer
        poster={currentLot?.imageUrl}
        anchorName={anchorName}
        onlineCount={room.snapshot?.onlineCount}
        wsState={wsState}
        roomName={roomName}
      />

      {clearScreen ? (
        <button type="button" className="exitClearScreen" onClick={() => setClearScreen(false)}>退出清屏</button>
      ) : null}
      <LiveRoomChrome controller={controller} />
      <LiveRoomEffectsLayer controller={controller} giftBurst={giftBurst} />
      <LiveBidLeaderboard
        controller={controller}
        onOpenAuction={openCurrentAuction}
        collapsed={leaderboardCollapsed}
        onToggleCollapsed={() => setLeaderboardCollapsed((value) => !value)}
      />
      {wsState !== '已连接' ? <div className="liveConnectionWarn">实时连接中断，正在恢复</div> : null}
      {error ? <div className="liveConnectionWarn error">{error}</div> : null}
      <AuctionDrawer controller={controller} />
      <AuctionNoticeLayer notices={notices} />
      <LiveComposer
        controller={controller}
        onOpenComments={() => setActiveSheet('comments')}
        onOpenAuction={openCurrentAuction}
        onOpenGift={() => setActiveSheet('gift')}
        onOpenMore={() => setActiveSheet('more')}
      />
      {activeSheet === 'comments' ? <DouyinCommentsSheet controller={controller} onClose={closeActiveSheet} /> : null}
      {activeSheet === 'share' ? <DouyinShareSheet controller={controller} onClose={closeActiveSheet} /> : null}
      {activeSheet === 'gift' ? <DouyinGiftSheet controller={controller} onClose={closeActiveSheet} onSendGift={sendGift} /> : null}
      {activeSheet === 'more' ? (
        <DouyinMoreSheet
          controller={controller}
          onClose={closeActiveSheet}
          onOpenAuction={openCurrentAuction}
          onOpenShare={() => setActiveSheet('share')}
          onToggleClear={() => setClearScreen(true)}
        />
      ) : null}

      {depositPrompt ? (
        <DepositPayModal
          lot={depositPrompt.lot}
          onConfirm={actions.confirmDepositPayment}
          onClose={actions.closeDepositPrompt}
        />
      ) : null}

      {resultLot ? (
        <ResultModal
          lot={resultLot}
          meId={meId}
          order={visibleResultOrder}
          onClose={actions.closeResult}
          onNext={actions.nextLot}
          onPay={actions.setPayOrder}
        />
      ) : null}

      {payOrder ? (
        <MockPayModal
          order={payOrder}
          onStartPayment={actions.markPaymentStarted}
          onPaid={actions.handlePaymentPaid}
          onClose={() => actions.setPayOrder(null)}
        />
      ) : null}
    </main>
  );
}
