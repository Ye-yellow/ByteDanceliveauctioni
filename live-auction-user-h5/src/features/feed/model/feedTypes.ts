export type FeedKind = 'video' | 'live';
export type HomeSheet = 'comment' | 'share' | 'more' | 'friend' | null;
export type CommentTarget = { videoId: string; comments: string };

export type FeedItem = {
  id: string;
  awemeId?: string;
  kind: FeedKind;
  author: string;
  avatarUrl?: string;
  title: string;
  music: string;
  location?: string;
  likes: string;
  comments: string;
  collects: string;
  shares: string;
  tone: 'rose' | 'cyan' | 'amber' | 'violet' | 'dark';
  videoUrl?: string;
  videoUrls?: string[];
  coverUrl?: string;
  sourceLabel?: string;
  liveLabel?: string;
  liveHref?: string;
  publisherHref?: string;
  mediaWidth?: number;
  mediaHeight?: number;
};

export type DouyinVideoRecord = {
  aweme_id?: string;
  desc?: string;
  author?: {
    nickname?: string;
    avatar_300x300?: { url_list?: string[] };
    avatar_168x168?: { url_list?: string[] };
    avatar_thumb?: { url_list?: string[] };
    cover_url?: Array<{ url_list?: string[] }>;
  };
  music?: {
    title?: string;
    author?: string;
  };
  video?: {
    play_addr?: { url_list?: string[] };
    cover?: { url_list?: string[] };
    width?: number;
    height?: number;
  };
  statistics?: {
    digg_count?: number;
    comment_count?: number;
    collect_count?: number;
    share_count?: number;
  };
};
