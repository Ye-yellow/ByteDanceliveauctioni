import { useEffect, useRef, useState, type FormEvent } from 'react';
import { DouyinBottomSheet } from './DouyinBottomSheet';

type RawDouyinComment = {
  comment_id?: string;
  create_time?: number;
  ip_location?: string;
  aweme_id?: string;
  content?: string;
  user_buried?: boolean;
  user_digged?: number | boolean;
  digg_count?: number | string;
  nickname?: string;
  avatar?: string;
  sub_comment_count?: number | string;
  replay?: string;
  children?: RawDouyinComment[];
};

type DouyinCommentReply = {
  id: string;
  createTime: number;
  ipLocation?: string;
  content: string;
  nickname: string;
  avatar?: string;
  userBuried: boolean;
  userDigged: boolean;
  diggCount: number;
  replyTo?: string;
};

type DouyinComment = DouyinCommentReply & {
  awemeId?: string;
  subCommentCount: number;
  showChildren: boolean;
  children: DouyinCommentReply[];
  replyEndReached: boolean;
};

type DouyinCommentSheetProps = {
  videoId?: string;
  onClose: () => void;
};

const COMMENT_BASE_URLS = ['/data/comments'];
const INITIAL_REPLY_PAGE_SIZE = 3;
const MORE_REPLY_PAGE_SIZE = 10;
const COMMENT_SOURCE_IDS = [
  '6686589698707590411',
  '6826943630775831812',
  '6882368275695586568',
  '6923214072347512068',
  '7000587983069957383',
  '7005490661592026405',
  '7110263965858549003',
  '7128686458763889956',
  '7161000281575148800',
  '7194815099381484860',
  '7260749400622894336',
  '7267478481213181238',
  '7270431418822446370',
  '7293100687989148943',
  '7295697246132227343',
  '7321200290739326262',
];

const COMMENT_FRIENDS = ['青柠汽水', '山海收藏家', '林间晚风', 'LiveAuction', 'Yexieer'];
const EMOJI_TOKENS = ['[流泪]', '[赞]', '[比心]', '[玫瑰]', '[笑哭]', '[看]'];

function countNumber(value: number | string | undefined) {
  if (typeof value === 'number') return Number.isFinite(value) ? value : 0;
  if (!value) return 0;
  const numeric = Number.parseInt(String(value), 10);
  return Number.isFinite(numeric) ? numeric : 0;
}

function formatCount(value: number) {
  if (value >= 100000000) return `${(value / 100000000).toFixed(value >= 1000000000 ? 0 : 1)}亿`;
  if (value >= 10000) return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}万`;
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`;
  return String(value);
}

function normalizeTime(value?: number) {
  if (!value) return Date.now();
  return value < 1000000000000 ? value * 1000 : value;
}

function formatCommentTime(value: number, location?: string) {
  const date = new Date(value);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}${location ? ` · ${location}` : ''}`;
}

function rawCommentId(raw: RawDouyinComment, index: number) {
  return raw.comment_id || `${raw.aweme_id || 'comment'}-${raw.create_time || 0}-${index}`;
}

function normalizeReply(raw: RawDouyinComment, index: number, parentId: string): DouyinCommentReply {
  return {
    id: `${parentId}-reply-${rawCommentId(raw, index)}`,
    createTime: normalizeTime(raw.create_time),
    ipLocation: raw.ip_location,
    content: raw.user_buried ? '该评论已折叠' : raw.content || '[图片表情]',
    nickname: raw.nickname || `抖音用户${index + 1}`,
    avatar: raw.avatar,
    userBuried: Boolean(raw.user_buried),
    userDigged: raw.user_digged === true || raw.user_digged === 1,
    diggCount: countNumber(raw.digg_count),
    replyTo: raw.replay,
  };
}

function normalizeComment(raw: RawDouyinComment, index: number): DouyinComment {
  const id = rawCommentId(raw, index);
  const children = Array.isArray(raw.children)
    ? raw.children.map((child, childIndex) => normalizeReply(child, childIndex, id))
    : [];
  const subCommentCount = Math.max(countNumber(raw.sub_comment_count), children.length);
  return {
    id,
    awemeId: raw.aweme_id,
    createTime: normalizeTime(raw.create_time),
    ipLocation: raw.ip_location,
    content: raw.user_buried ? '该评论已折叠' : raw.content || '[图片表情]',
    nickname: raw.nickname || `抖音用户${index + 1}`,
    avatar: raw.avatar,
    userBuried: Boolean(raw.user_buried),
    userDigged: raw.user_digged === true || raw.user_digged === 1,
    diggCount: countNumber(raw.digg_count),
    subCommentCount,
    showChildren: false,
    children,
    replyEndReached: false,
  };
}

function extractRows(payload: unknown): RawDouyinComment[] {
  if (Array.isArray(payload)) return payload as RawDouyinComment[];
  if (payload && typeof payload === 'object') {
    const row = payload as { comments?: unknown; data?: unknown; list?: unknown };
    if (Array.isArray(row.comments)) return row.comments as RawDouyinComment[];
    if (Array.isArray(row.data)) return row.data as RawDouyinComment[];
    if (Array.isArray(row.list)) return row.list as RawDouyinComment[];
  }
  return [];
}

async function loadCommentRows(videoId?: string) {
  const ids = commentSourceOrder(videoId);
  for (const id of ids) {
    const rows = await fetchCommentRows(id);
    if (rows.length) return rows;
  }
  return [];
}

function stableHash(value: string) {
  let hash = 0;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) >>> 0;
  }
  return hash;
}

function commentSourceOrder(videoId?: string) {
  if (!videoId) return COMMENT_SOURCE_IDS;
  const sourceSet = new Set(COMMENT_SOURCE_IDS);
  const fallbackStart = stableHash(videoId) % COMMENT_SOURCE_IDS.length;
  const fallbackIds = Array.from({ length: COMMENT_SOURCE_IDS.length }, (_, index) => (
    COMMENT_SOURCE_IDS[(fallbackStart + index) % COMMENT_SOURCE_IDS.length]
  ));
  return sourceSet.has(videoId) ? [videoId, ...fallbackIds.filter((id) => id !== videoId)] : [videoId, ...fallbackIds];
}

async function fetchCommentRows(id: string, timeoutMs = 6000) {
  for (const baseUrl of COMMENT_BASE_URLS) {
    const controller = new AbortController();
    const timer = window.setTimeout(() => controller.abort(), timeoutMs);
    try {
      const response = await fetch(`${baseUrl}/video_id_${id}.json`, baseUrl.startsWith('/') ? { signal: controller.signal } : { mode: 'cors', signal: controller.signal });
      if (response.ok) {
        const rows = extractRows(await response.json());
        if (rows.length) return rows;
      }
    } catch {
      // Try the next source.
    } finally {
      window.clearTimeout(timer);
    }
  }
  return [];
}

function commentToRaw(comment: DouyinComment): RawDouyinComment {
  return {
    comment_id: comment.id,
    create_time: Math.floor(comment.createTime / 1000),
    ip_location: comment.ipLocation,
    aweme_id: comment.awemeId,
    content: comment.content,
    user_buried: comment.userBuried,
    user_digged: comment.userDigged,
    digg_count: comment.diggCount,
    nickname: comment.nickname,
    avatar: comment.avatar,
    sub_comment_count: comment.subCommentCount,
  };
}

function buildReplies(base: DouyinComment, comments: DouyinComment[], pageSize: number) {
  const existingIds = new Set([base.id, ...base.children.map((reply) => reply.id)]);
  const candidates = comments
    .filter((comment) => comment.id !== base.id)
    .map(commentToRaw);
  if (!candidates.length) return [];
  const start = (stableHash(base.id) + base.children.length) % candidates.length;
  const replies: DouyinCommentReply[] = [];
  for (let step = 0; step < candidates.length && replies.length < pageSize; step += 1) {
    const sourceIndex = (start + step) % candidates.length;
    const reply = normalizeReply(candidates[sourceIndex], sourceIndex, base.id);
    if (!existingIds.has(reply.id)) {
      existingIds.add(reply.id);
      replies.push(reply);
    }
  }
  return replies;
}

function replyButtonText(comment: DouyinComment, loading: boolean) {
  if (loading) return '加载回复中';
  if (!comment.showChildren) return `展开${comment.subCommentCount || comment.children.length || INITIAL_REPLY_PAGE_SIZE}条回复`;
  if (comment.replyEndReached) return '已展示全部回复';
  return '展开更多回复';
}

function placeCaretAtEnd(element: HTMLElement) {
  element.focus();
  const range = document.createRange();
  range.selectNodeContents(element);
  range.collapse(false);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
}

function HeartIcon({ filled = false }: { filled?: boolean }) {
  return (
    <svg viewBox="0 0 48 48" aria-hidden="true">
      <path
        fill={filled ? 'currentColor' : 'none'}
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="4"
        d="M15 8C8.925 8 4 12.925 4 19c0 11 13 21 20 23.326C31 40 44 30 44 19c0-6.075-4.925-11-11-11c-3.72 0-7.01 1.847-9 4.674A10.99 10.99 0 0 0 15 8"
      />
    </svg>
  );
}

function BuryIcon() {
  return (
    <svg viewBox="0 0 48 48" aria-hidden="true">
      <path
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="4"
        d="m24 31-3-5 7-6-9-5 1-5.8C18.5 8.432 16.8 8 15 8C8.925 8 4 12.925 4 19c0 11 13 21 20 23 7-2 20-12 20-23 0-6.075-4.925-11-11-11-1.8 0-3.5.433-5 1.2"
      />
    </svg>
  );
}

function Avatar({ name, src }: { name: string; src?: string }) {
  const [failed, setFailed] = useState(false);
  return (
    <span className="dyCommentAvatar">
      {src && !failed ? (
        <img src={src} alt="" loading="lazy" referrerPolicy="no-referrer" onError={() => setFailed(true)} />
      ) : (
        <span className="dyCommentAvatarPlaceholder">{name.slice(0, 1)}</span>
      )}
    </span>
  );
}

function CommentThread({
  comment,
  loadingReply,
  onLike,
  onBury,
  onReply,
  onToggleReplies,
}: {
  comment: DouyinComment;
  loadingReply: boolean;
  onLike: () => void;
  onBury: () => void;
  onReply: () => void;
  onToggleReplies: () => void;
}) {
  return (
    <article className="dyCommentItem dyCommentThread">
      <div className="dyCommentMain">
        <Avatar name={comment.nickname} src={comment.avatar} />
        <div className="dyCommentBody">
          <div className="dyCommentName">{comment.nickname}</div>
          <p className={comment.userBuried ? 'dyCommentText folded' : 'dyCommentText'}>{comment.userBuried ? '该评论已折叠' : comment.content}</p>
          <div className="dyCommentMeta">
            <div className="dyCommentMetaLeft">
              <span className="dyCommentTime">{formatCommentTime(comment.createTime, comment.ipLocation)}</span>
              <button type="button" className="dyCommentReply" onClick={onReply}>回复</button>
            </div>
            <div className="dyCommentMetaRight">
              <button
                type="button"
                className={`dyCommentVote${comment.userDigged ? ' active' : ''}`}
                aria-label="点赞评论"
                onClick={onLike}
              >
                <HeartIcon filled={comment.userDigged} />
                <span>{comment.diggCount ? formatCount(comment.diggCount) : ''}</span>
              </button>
              <button type="button" className={`dyCommentVote dyCommentBury${comment.userBuried ? ' active' : ''}`} aria-label="不喜欢评论" onClick={onBury}>
                <BuryIcon />
              </button>
            </div>
          </div>
          {comment.subCommentCount > 0 || comment.children.length ? (
            <div className="dyCommentReplies">
              {comment.showChildren ? (
                <div className="dyCommentReplyList">
                  {comment.children.map((reply) => (
                    <div className="dyCommentReplyItem" key={reply.id}>
                      <Avatar name={reply.nickname} src={reply.avatar} />
                      <div>
                        <b>{reply.nickname}{reply.replyTo ? <span className="dyCommentReplyTarget"> 回复 {reply.replyTo}</span> : null}</b>
                        <p>{reply.content}</p>
                        <span>{formatCommentTime(reply.createTime, reply.ipLocation)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : null}
              <button type="button" className="dyCommentExpand" onClick={onToggleReplies} disabled={comment.showChildren && comment.replyEndReached}>
                <span className="dyCommentExpandLine" />
                <span>{replyButtonText(comment, loadingReply)}</span>
                <svg viewBox="0 0 1024 1024"><path fill="currentColor" d="M104.7 338.8a64 64 0 0 1 90.5 0L512 655.6l316.8-316.8a64 64 0 0 1 90.5 90.4l-362 362.1a64 64 0 0 1-90.5 0l-362.1-362a64 64 0 0 1 0-90.5" /></svg>
              </button>
            </div>
          ) : null}
        </div>
      </div>
    </article>
  );
}

export function DouyinCommentSheet({ videoId, onClose }: DouyinCommentSheetProps) {
  const [comments, setComments] = useState<DouyinComment[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingReplyId, setLoadingReplyId] = useState('');
  const [draft, setDraft] = useState('');
  const [friendOpen, setFriendOpen] = useState(false);
  const editorRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let disposed = false;
    void loadCommentRows(videoId).then((rows) => {
      if (disposed) return;
      setComments(rows.map(normalizeComment));
      setLoading(false);
    });
    return () => {
      disposed = true;
    };
  }, [videoId]);

  const titleText = loading ? '评论' : `${formatCount(comments.length)}条评论`;

  function setEditorText(value: string) {
    setDraft(value);
    if (editorRef.current) {
      editorRef.current.textContent = value;
      window.requestAnimationFrame(() => {
        if (editorRef.current) placeCaretAtEnd(editorRef.current);
      });
    }
  }

  function appendToken(value: string) {
    const next = draft ? `${draft}${value}` : value;
    setEditorText(next);
  }

  function handleInput(event: FormEvent<HTMLDivElement>) {
    setDraft(event.currentTarget.textContent || '');
  }

  function handleReply(comment: DouyinComment) {
    setEditorText(`回复 @${comment.nickname} `);
  }

  function handleLike(commentId: string) {
    setComments((current) => current.map((comment) => {
      if (comment.id !== commentId) return comment;
      const liked = !comment.userDigged;
      return {
        ...comment,
        userDigged: liked,
        diggCount: Math.max(0, comment.diggCount + (liked ? 1 : -1)),
      };
    }));
  }

  function handleBury(commentId: string) {
    setComments((current) => current.map((comment) => (
      comment.id === commentId ? { ...comment, userBuried: !comment.userBuried } : comment
    )));
  }

  function handleToggleReplies(commentId: string) {
    const target = comments.find((comment) => comment.id === commentId);
    if (!target || loadingReplyId || target.replyEndReached) return;
    if (target.children.length && !target.showChildren) {
      setComments((current) => current.map((comment) => (
        comment.id === commentId ? { ...comment, showChildren: true } : comment
      )));
      return;
    }
    setLoadingReplyId(commentId);
    window.setTimeout(() => {
      setComments((current) => current.map((comment) => {
        if (comment.id !== commentId) return comment;
        const remainingCount = comment.subCommentCount
          ? Math.max(0, comment.subCommentCount - comment.children.length)
          : INITIAL_REPLY_PAGE_SIZE;
        const pageSize = comment.showChildren ? MORE_REPLY_PAGE_SIZE : INITIAL_REPLY_PAGE_SIZE;
        const requestedSize = Math.min(pageSize, remainingCount || pageSize);
        const nextReplies = buildReplies(comment, current, requestedSize);
        const nextCount = comment.children.length + nextReplies.length;
        return {
          ...comment,
          showChildren: true,
          children: [...comment.children, ...nextReplies],
          replyEndReached: nextReplies.length < requestedSize
            || (comment.subCommentCount > 0 && nextCount >= comment.subCommentCount),
        };
      }));
      setLoadingReplyId('');
    }, 260);
  }

  function handleSend() {
    const text = draft.trim();
    if (!text) return;
    const localComment: DouyinComment = {
      id: `local-${Date.now()}`,
      createTime: Date.now(),
      content: text,
      nickname: '我',
      userBuried: false,
      userDigged: false,
      diggCount: 0,
      subCommentCount: 0,
      showChildren: false,
      children: [],
      replyEndReached: false,
    };
    setComments((current) => [localComment, ...current]);
    setEditorText('');
    setFriendOpen(false);
  }

  return (
    <DouyinBottomSheet label="评论" className="dyReplicaCommentSheet" height="70%" maskMode="light" onClose={onClose}>
      {({ close, scrollRef }) => (
        <>
          <header className="dyCommentHeader">
            <button type="button" className="dyCommentCloseBtn dyCommentGhostClose" aria-label="返回" onClick={close}>‹</button>
            <span className="dyCommentTitle">{titleText}</span>
            <div className="dyCommentHeaderRight">
              <button type="button" aria-label="全屏">
                <svg viewBox="0 0 24 24"><path fill="currentColor" fillRule="evenodd" d="M14 3.75a.75.75 0 0 0 0 1.5h3.69l-4.72 4.72a.75.75 0 1 0 1.06 1.06l4.72-4.72V10a.75.75 0 0 0 1.5 0V4.5a.75.75 0 0 0-.75-.75zm-4 16.5a.75.75 0 0 0 0-1.5H6.31l4.72-4.72a.75.75 0 1 0-1.06-1.06l-4.72 4.72V14a.75.75 0 0 0-1.5 0v5.5c0 .414.336.75.75.75z" clipRule="evenodd" /></svg>
              </button>
              <button type="button" aria-label="关闭" onClick={close}>
                <svg viewBox="0 0 24 24"><path fill="currentColor" d="M18.3 5.71a.996.996 0 0 0-1.41 0L12 10.59 7.11 5.7A.996.996 0 1 0 5.7 7.11L10.59 12 5.7 16.89a.996.996 0 1 0 1.41 1.41L12 13.41l4.89 4.89a.996.996 0 1 0 1.41-1.41L13.41 12l4.89-4.89c.38-.38.38-1.02 0-1.4" /></svg>
              </button>
            </div>
          </header>

          <div className="dyCommentList" ref={scrollRef}>
            {loading ? (
              <div className="dyCommentLoading">评论加载中...</div>
            ) : (
              <>
                {comments.map((comment) => (
                  <CommentThread
                    key={comment.id}
                    comment={comment}
                    loadingReply={loadingReplyId === comment.id}
                    onLike={() => handleLike(comment.id)}
                    onBury={() => handleBury(comment.id)}
                    onReply={() => handleReply(comment)}
                    onToggleReplies={() => handleToggleReplies(comment.id)}
                  />
                ))}
                <div className="dyCommentNoMore">暂时没有更多了</div>
              </>
            )}
          </div>

          <div className="dyCommentToolbar">
            {friendOpen ? (
              <div className="dyCommentFriendStrip" aria-label="@朋友">
                {COMMENT_FRIENDS.map((name) => (
                  <button type="button" className="dyCommentFriend" onClick={() => appendToken(`@${name} `)} key={name}>
                    <Avatar name={name} />
                    <span>{name}</span>
                  </button>
                ))}
              </div>
            ) : null}
            <div className="dyCommentToolbarLine">
              <div
                ref={editorRef}
                className="dyCommentEditor"
                contentEditable
                suppressContentEditableWarning
                role="textbox"
                aria-label="发表评论"
                data-placeholder="善语结善缘，恶言伤人心"
                onInput={handleInput}
              />
              <div className="dyCommentToolbarActions">
                <button type="button" className="dyCommentToolBtn" aria-label="@提及" onClick={() => setFriendOpen((value) => !value)}>@</button>
                <button type="button" className="dyCommentToolBtn" aria-label="表情" onClick={() => appendToken(EMOJI_TOKENS[Math.floor(Math.random() * EMOJI_TOKENS.length)])}>☺</button>
                {draft.trim() ? <button type="button" className="dyCommentSendButton" onClick={handleSend}>发送</button> : null}
              </div>
            </div>
          </div>
        </>
      )}
    </DouyinBottomSheet>
  );
}
