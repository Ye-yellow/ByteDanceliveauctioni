import { useState } from 'react';

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home/report');
}

export function SubmitReportPage() {
  const params = new URLSearchParams(location.search);
  const [desc, setDesc] = useState('');
  const [photos, setPhotos] = useState<string[]>([]);
  const [submitted, setSubmitted] = useState(false);
  const reason = params.get('type') || '色情低俗';
  const mode = params.get('mode') || 'video';
  const title = mode === 'music' ? '音乐举报' : mode === 'chat' ? '私信举报' : '视频举报';

  return (
    <main className="mobileShell dyReportReplicaPage">
      <header className="dyReportReplicaHeader">
        <button type="button" aria-label="返回" onClick={goBack}>‹</button>
        <h1>{title}</h1>
        <span />
      </header>
      <section className="dySubmitReportContent">
        <p>你的举报将在12小时内受理，若举报成功会第一时间告知处理结果，请尽量提交完整的举报描述及材料</p>
        <section className="dySubmitReason">举报理由：{reason}</section>
        <label className="dySubmitTextarea">
          <span>举报描述(选填)</span>
          <textarea maxLength={200} value={desc} placeholder="详细描述举报理由" onChange={(event) => setDesc(event.target.value)} />
          <small>{desc.length}/200</small>
        </label>
        <section className="dySubmitPhotos">
          {photos.map((photo) => (
            <button type="button" key={photo} onClick={() => setPhotos((items) => items.filter((item) => item !== photo))}>
              <span>{photo}</span>
              <i>×</i>
            </button>
          ))}
          {photos.length < 4 ? (
            <button type="button" onClick={() => setPhotos((items) => [...items, `${items.length + 1}/4`])}>
              <b>▣</b>
              <span>{photos.length}/4</span>
            </button>
          ) : null}
        </section>
        <button className="dySubmitButton" type="button" onClick={() => setSubmitted(true)}>提交</button>
        {submitted ? <em className="dySubmitToast">已提交</em> : null}
      </section>
    </main>
  );
}
