import { useCallback, useEffect, useRef, useState, type CSSProperties, type ReactNode, type RefObject, type TouchEvent } from 'react';

type DouyinBottomSheetProps = {
  label: string;
  className?: string;
  height?: string;
  maskMode?: 'light' | 'dark';
  onClose: () => void;
  children: (api: { close: () => void; scrollRef: RefObject<HTMLDivElement | null> }) => ReactNode;
};

const DEFAULT_HEIGHT = 'calc(var(--vh, 1vh) * 70)';
const CLOSE_DURATION_MS = 250;

export function DouyinBottomSheet({
  label,
  className = '',
  height = DEFAULT_HEIGHT,
  maskMode = 'light',
  onClose,
  children,
}: DouyinBottomSheetProps) {
  const [translateY, setTranslateY] = useState(0);
  const [closing, setClosing] = useState(false);
  const [dragging, setDragging] = useState(false);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const sheetRef = useRef<HTMLElement | null>(null);
  const touchStart = useRef({ y: 0, at: 0 });
  const closeTimer = useRef<number | undefined>(undefined);

  const requestClose = useCallback(() => {
    if (closing) return;
    const sheetHeight = sheetRef.current?.getBoundingClientRect().height || window.innerHeight * 0.7;
    setClosing(true);
    setDragging(false);
    setTranslateY(sheetHeight + 8);
    closeTimer.current = window.setTimeout(onClose, CLOSE_DURATION_MS);
  }, [closing, onClose]);

  useEffect(() => {
    const previous = {
      overflow: document.body.style.overflow,
      position: document.body.style.position,
      top: document.body.style.top,
      width: document.body.style.width,
    };
    const scrollY = window.scrollY;
    document.body.style.overflow = 'hidden';
    document.body.style.position = 'fixed';
    document.body.style.top = `-${scrollY}px`;
    document.body.style.width = '100%';
    return () => {
      if (closeTimer.current) window.clearTimeout(closeTimer.current);
      document.body.style.overflow = previous.overflow;
      document.body.style.position = previous.position;
      document.body.style.top = previous.top;
      document.body.style.width = previous.width;
      window.scrollTo(0, scrollY);
    };
  }, []);

  function handleTouchStart(event: TouchEvent<HTMLElement>) {
    event.stopPropagation();
    if ((scrollRef.current?.scrollTop || 0) > 0) {
      setDragging(false);
      return;
    }
    const touch = event.touches[0];
    touchStart.current = { y: touch?.clientY ?? 0, at: Date.now() };
    setDragging(true);
  }

  function handleTouchMove(event: TouchEvent<HTMLElement>) {
    event.stopPropagation();
    if (!dragging) return;
    const touch = event.touches[0];
    const dy = Math.max(0, (touch?.clientY ?? touchStart.current.y) - touchStart.current.y);
    if (dy > 0) {
      event.preventDefault();
      setTranslateY(dy);
    }
  }

  function handleTouchEnd(event: TouchEvent<HTMLElement>) {
    event.stopPropagation();
    if (!dragging) return;
    const touch = event.changedTouches[0];
    const dy = Math.max(0, (touch?.clientY ?? touchStart.current.y) - touchStart.current.y);
    const elapsed = Date.now() - touchStart.current.at;
    setDragging(false);
    const sheetHeight = sheetRef.current?.getBoundingClientRect().height || window.innerHeight * 0.7;
    if (dy > sheetHeight * 0.5 || (elapsed < 240 && dy > 92)) {
      requestClose();
      return;
    }
    setTranslateY(0);
  }

  const classNames = ['dyBottomSheet', className, dragging ? 'isDragging' : '', closing ? 'isClosing' : '']
    .filter(Boolean)
    .join(' ');
  const style = {
    '--dy-bottom-sheet-height': height,
    transform: `translate3d(0, ${translateY}px, 0)`,
  } as CSSProperties;
  const maskStyle = {
    '--dy-bottom-sheet-height': height,
  } as CSSProperties;

  return (
    <>
      <button type="button" className={`dyBottomSheetMask ${maskMode}`} aria-label="关闭弹层" style={maskStyle} onClick={requestClose} />
      <section
        ref={sheetRef}
        className={classNames}
        role="dialog"
        aria-modal="true"
        aria-label={label}
        style={style}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
        onTouchCancel={handleTouchEnd}
      >
        {children({ close: requestClose, scrollRef })}
      </section>
    </>
  );
}
