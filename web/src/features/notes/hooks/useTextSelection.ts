// features/notes/hooks/useTextSelection.ts
import { useEffect, useState, type RefObject } from "react";

export type SelectionInfo = { text: string; rect: DOMRect; range: Range };

export function useTextSelection(containerRef: RefObject<HTMLElement | null>) {
  const [selection, setSelection] = useState<SelectionInfo | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    function read() {
      const sel = window.getSelection();
      if (!sel || sel.isCollapsed || sel.rangeCount === 0) {
        setSelection(null);
        return;
      }
      const range = sel.getRangeAt(0);
      const text = sel.toString().trim();
      if (!text || !container!.contains(range.commonAncestorContainer)) {
        setSelection(null);
        return;
      }
      setSelection({
        text,
        rect: range.getBoundingClientRect(),
        range: range.cloneRange(),
      });
    }

    // setTimeout so the browser finishes updating the selection first
    const onUp = () => setTimeout(read, 0);

    // keep the toolbar attached while the notes area scrolls
    const onScroll = () =>
      setSelection((s) =>
        s ? { ...s, rect: s.range.getBoundingClientRect() } : s
      );

    document.addEventListener("mouseup", onUp);
    document.addEventListener("keyup", onUp);
    document.addEventListener("scroll", onScroll, true); // capture: catches inner scroll containers

    return () => {
      document.removeEventListener("mouseup", onUp);
      document.removeEventListener("keyup", onUp);
      document.removeEventListener("scroll", onScroll, true);
    };
  }, [containerRef]);

  const clear = () => {
    window.getSelection()?.removeAllRanges();
    setSelection(null);
  };

  return { selection, clear };
}