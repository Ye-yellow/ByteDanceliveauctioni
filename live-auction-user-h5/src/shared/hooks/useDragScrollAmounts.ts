import { useEffect } from 'react';

export function useDragScrollAmounts() {
  useEffect(() => {
    let active: HTMLElement | null = null;
    let startX = 0;
    let startScrollLeft = 0;
    let pointerId = 0;

    const reset = () => {
      active?.classList.remove('isDragging');
      active = null;
      pointerId = 0;
    };

    const onPointerDown = (event: PointerEvent) => {
      if (event.pointerType === 'touch' || (event.pointerType === 'mouse' && event.button !== 0)) return;
      const target = (event.target as Element | null)?.closest<HTMLElement>('.scrollAmount');
      if (!target || target.scrollWidth <= target.clientWidth) return;
      active = target;
      pointerId = event.pointerId;
      startX = event.clientX;
      startScrollLeft = target.scrollLeft;
      target.classList.add('isDragging');
      target.setPointerCapture?.(event.pointerId);
    };

    const onPointerMove = (event: PointerEvent) => {
      if (!active || event.pointerId !== pointerId) return;
      const deltaX = event.clientX - startX;
      if (Math.abs(deltaX) > 2) event.preventDefault();
      active.scrollLeft = startScrollLeft - deltaX;
    };

    const onPointerUp = (event: PointerEvent) => {
      if (!active || event.pointerId !== pointerId) return;
      active.releasePointerCapture?.(event.pointerId);
      reset();
    };

    document.addEventListener('pointerdown', onPointerDown);
    document.addEventListener('pointermove', onPointerMove, { passive: false });
    document.addEventListener('pointerup', onPointerUp);
    document.addEventListener('pointercancel', onPointerUp);

    return () => {
      document.removeEventListener('pointerdown', onPointerDown);
      document.removeEventListener('pointermove', onPointerMove);
      document.removeEventListener('pointerup', onPointerUp);
      document.removeEventListener('pointercancel', onPointerUp);
    };
  }, []);
}
