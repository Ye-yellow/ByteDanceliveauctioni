const REPORT_GROUPS = [
  ['内容违规', ['色情低俗', '时政不实信息', '违法犯罪', '垃圾广告、售卖假货等', '造谣传播', '涉嫌欺诈', '侮辱漫骂', '危险行为', '涉嫌非法集资', '价值观导向不良']],
  ['侵犯名誉', ['侵犯名誉、隐私、肖像权等', '内容盗用本人作品', '内容盗用他人作品']],
  ['未成年', ['未成年人不当行为', '内容不适合未成年观看']],
  ['其他', ['引人不适', '疑似自我伤害', '诱导点赞、分享、关注', '其他']],
] as const;

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/video-detail');
}

function modeTitle() {
  const mode = new URLSearchParams(location.search).get('mode') || 'video';
  if (mode === 'music') return '音乐举报';
  if (mode === 'chat') return '私信举报';
  return '视频举报';
}

export function ReportPage() {
  const mode = new URLSearchParams(location.search).get('mode') || 'video';
  return (
    <main className="mobileShell dyReportReplicaPage">
      <header className="dyReportReplicaHeader">
        <button type="button" aria-label="返回" onClick={goBack}>‹</button>
        <h1>{modeTitle()}</h1>
        <span />
      </header>
      <section className="dyReportReplicaContent">
        {REPORT_GROUPS.map(([group, rows]) => (
          <section className="dyReportReplicaGroup" key={group}>
            <h2>{group}</h2>
            {rows.map((row) => (
              <a href={`/home/submit-report?type=${encodeURIComponent(row)}&mode=${encodeURIComponent(mode)}`} key={row}>
                <span>{row}</span>
                <i>›</i>
              </a>
            ))}
          </section>
        ))}
      </section>
    </main>
  );
}
