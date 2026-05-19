export type Money = number;

export type Bid = {
  id: string;
  lotId: string;
  userId: string;
  nickname: string;
  amount: Money;
  createdAt: string;
};

export type RankingItem = {
  rank: number;
  userId: string;
  nickname: string;
  amount: Money;
  at: string;
};

export type Lot = {
  id: string;
  roomId: string;
  sessionId?: string;
  title: string;
  description: string;
  imageUrl: string;
  startPrice: Money;
  currentPrice: Money;
  minIncrement: Money;
  status: string;
  endsAt: string;
  winnerUserId?: string;
  version: number;
  bids: Bid[];
  ranking?: RankingItem[];
  atmosphereText?: string;
};

export type PlaybookStage =
  | 'WARM_UP'
  | 'TRUST_BLOCKED'
  | 'BIDDING_ACTIVE'
  | 'DUEL_READY'
  | 'DUEL_MODE'
  | 'COOLING'
  | 'SETTLE_READY';

export type TrustRevealCard = {
  id: string;
  type: 'CERTIFICATE' | 'FLAW' | 'DETAIL' | 'PRICE_REF' | 'SERVICE';
  title: string;
  content: string;
  revealed: boolean;
};
