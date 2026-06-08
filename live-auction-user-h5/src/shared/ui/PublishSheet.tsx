type PublishSheetProps = {
  roomHref?: string;
  historyHref?: string;
  onClose: () => void;
};

const PUBLISH_ACTIONS = [
  { title: '拍品开箱', desc: '发布图文和短视频，沉淀到主页作品' },
  { title: '竞拍预告', desc: '提前告知下一场直播拍品和时间' },
  { title: '讲解卡', desc: '补充材质、瑕疵、尺寸和亮点' },
  { title: '成交晒单', desc: '展示竞拍结果和买家反馈' },
];

export function PublishSheet({ roomHref, historyHref, onClose }: PublishSheetProps) {
  return (
    <div
      className="douyinSheetMask dark publishSheetMask"
      onClick={onClose}
      onTouchStart={(event) => event.stopPropagation()}
      onTouchEnd={(event) => event.stopPropagation()}
    >
      <section className="douyinBottomSheet publishSheet" role="dialog" aria-modal="true" aria-label="发布内容" onClick={(event) => event.stopPropagation()}>
        <header className="publishSheetHeader">
          <b>发布</b>
          <button type="button" className="sheetClose" onClick={onClose} aria-label="关闭发布面板">×</button>
        </header>

        <section className="publishCameraCard" aria-label="快速发布">
          <div>
            <span>+</span>
          </div>
          <b>开始创作</b>
          <p>拍品开箱、竞拍预告、讲解卡都可以从这里开始。</p>
        </section>

        <div className="publishActionGrid">
          {PUBLISH_ACTIONS.map((item) => (
            <button type="button" key={item.title}>
              <b>{item.title}</b>
              <small>{item.desc}</small>
            </button>
          ))}
        </div>

        <footer className="publishSheetLinks">
          {roomHref ? <a href={roomHref}>进入当前直播间</a> : null}
          {historyHref ? <a href={historyHref}>查看我的订单</a> : null}
        </footer>
      </section>
    </div>
  );
}
