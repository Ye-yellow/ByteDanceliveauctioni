export type RoomVisualProfile = {
  roomName: string;
  anchorName: string;
  heatText: string;
  summary: string;
};

export const ROOM_VISUAL_PROFILES: RoomVisualProfile[] = [
  {
    roomName: '直播竞拍',
    anchorName: '主播团队',
    heatText: '直播竞拍热度上升中',
    summary: '好物上新，进直播间参与竞拍',
  },
  {
    roomName: '收藏专场',
    anchorName: '收藏顾问',
    heatText: '收藏榜热度上升中',
    summary: '精选拍品轮番开拍',
  },
];

export const DEFAULT_ROOM_VISUAL_PROFILE = ROOM_VISUAL_PROFILES[0];

export function roomVisualProfileAt(index: number): RoomVisualProfile {
  return ROOM_VISUAL_PROFILES[index % ROOM_VISUAL_PROFILES.length] || DEFAULT_ROOM_VISUAL_PROFILE;
}
