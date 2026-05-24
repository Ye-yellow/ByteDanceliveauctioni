export type DemoRoomProfile = {
  roomId: string;
  roomName: string;
  anchorName: string;
  heatText: string;
  summary: string;
};

export const DEMO_ROOM_PROFILES: DemoRoomProfile[] = [
  {
    roomId: 'room-jewel-01',
    roomName: '珠宝竞拍直播',
    anchorName: '粤海大专',
    heatText: '拍卖主播热度第 29 名',
    summary: '今晚好物上新，进直播间参与竞拍',
  },
  {
    roomId: 'room-jade-02',
    roomName: '玉石严选直播',
    anchorName: '@翠藏主理人',
    heatText: '玉石榜热度第 8 名',
    summary: '手镯、挂件、原石专场正在上新',
  },
  {
    roomId: 'room-watch-03',
    roomName: '腕表收藏直播',
    anchorName: '@钟表档口老周',
    heatText: '腕表榜热度第 12 名',
    summary: '机械表、古董表、限量款轮番开拍',
  },
  {
    roomId: 'room-tea-04',
    roomName: '茶器雅集直播',
    anchorName: '@器物研究所',
    heatText: '文玩榜热度第 16 名',
    summary: '紫砂、银壶、老茶具今晚连续竞拍',
  },
];

export const DEFAULT_DEMO_ROOM_PROFILE = DEMO_ROOM_PROFILES[0];

export function getDemoRoomProfile(roomId: string): DemoRoomProfile | undefined {
  return DEMO_ROOM_PROFILES.find((room) => room.roomId === roomId);
}
