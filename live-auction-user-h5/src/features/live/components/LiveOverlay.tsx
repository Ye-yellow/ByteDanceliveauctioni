export function LiveOverlay({
  anchorName,
  onlineCount,
  wsState,
  roomName,
}: {
  anchorName?: string;
  onlineCount?: number;
  wsState: string;
  roomName: string;
}) {
  return (
    <div className="liveOverlay">
      <div className="liveTopLine">
        <span className="liveBadge">LIVE</span>
        <b>{anchorName || "主播"}</b>
        <span
          className={`wsDot ${wsState === "已连接" ? "ok" : wsState === "重连中" ? "warn" : "off"}`}
        >
          {wsState}
        </span>
      </div>
      <div className="liveHeat">
        <span>
          🔥 {Math.max(onlineCount || 0, 1).toLocaleString("zh-CN")} 在线
        </span>
        <i>+99</i>
        <i>❤️</i>
        <i>竞拍热度上升</i>
      </div>
      <div className="liveRoomName">{roomName}</div>
    </div>
  );
}
