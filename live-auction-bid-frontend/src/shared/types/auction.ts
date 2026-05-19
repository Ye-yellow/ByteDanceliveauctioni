export type { Bid, BidRule, CreateLotRequest, Lot, Money, PlaceBidRequest, RankingItem } from '../api/types';

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
