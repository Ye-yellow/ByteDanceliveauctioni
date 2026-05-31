export function AuctionNoticeLayer({ notices }: { notices: string[] }) {
  return (
    <div className="noticeLayer" aria-live="polite">
      {notices.slice(0, 4).map((notice, index) => (
        <span key={`${notice}-${index}`}>{notice}</span>
      ))}
    </div>
  );
}
