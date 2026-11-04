import { useEffect, type RefObject } from "react";

const FOLD: Record<string, string> = {
  "’": "'", "‘": "'", "“": '"', "”": '"', "–": "-", "—": "-",
};
const fold = (ch: string) => (FOLD[ch] ?? ch).toLowerCase();

function normalizeNeedle(needle: string): string {
  let target = "";
  let prev = true;
  for (const ch of needle) {
    const sp = /\s/.test(ch);
    if (sp && prev) continue;
    target += sp ? " " : fold(ch);
    prev = sp;
  }
  return target.trim();
}

interface TextMap {
  hay: string;
  map: { node: Text; offset: number }[];
}

function buildTextMap(root: HTMLElement): TextMap {
  let hay = "";
  const map: { node: Text; offset: number }[] = [];
  let prevSpace = true;
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  for (let n = walker.nextNode() as Text | null; n; n = walker.nextNode() as Text | null) {
    if (!prevSpace) {
      hay += " ";
      map.push({ node: n, offset: 0 });
      prevSpace = true;
    }
    for (let i = 0; i < n.data.length; i++) {
      const isSpace = /\s/.test(n.data[i]);
      if (isSpace && prevSpace) continue;
      hay += isSpace ? " " : fold(n.data[i]);
      map.push({ node: n, offset: i });
      prevSpace = isSpace;
    }
  }
  return { hay, map };
}

export function findAllRanges(root: HTMLElement, texts: string[]): Range[] {
  const needles = Array.from(
    new Set(texts.map(normalizeNeedle).filter((t) => t.length > 0)),
  );
  if (needles.length === 0) return [];

  const { hay, map } = buildTextMap(root);
  const ranges: Range[] = [];

  for (const target of needles) {
    let from = 0;
    for (;;) {
      const idx = hay.indexOf(target, from);
      if (idx < 0) break;
      const s = map[idx];
      const e = map[idx + target.length - 1];
      const r = new Range();
      r.setStart(s.node, s.offset);
      r.setEnd(e.node, e.offset + 1);
      ranges.push(r);
      from = idx + target.length;
    }
  }
  return ranges;
}

export function findRanges(root: HTMLElement, needle: string): Range[] {
  return findAllRanges(root, [needle]);
}

export function useRangeHighlight(
  containerRef: RefObject<HTMLElement | null>,
  texts: string[],
  name = "ref-hover",
) {
  const key = texts.join("\u0000");
  useEffect(() => {
    if (!("highlights" in CSS)) return; // unsupported browser: no-op
    const el = containerRef.current;
    if (!el || texts.length === 0) {
      CSS.highlights.delete(name);
      return;
    }
    const ranges = findAllRanges(el, texts);
    if (ranges.length === 0) {
      CSS.highlights.delete(name);
      return;
    }
    CSS.highlights.set(name, new Highlight(...ranges));
    return () => {
      CSS.highlights.delete(name);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, name, containerRef]);
}

function getScrollParent(el: Element | null): HTMLElement | null {
  for (let n = el; n; n = n.parentElement) {
    const { overflowY } = getComputedStyle(n);
    if ((overflowY === "auto" || overflowY === "scroll") && n.scrollHeight > n.clientHeight) {
      return n as HTMLElement;
    }
  }
  return null;
}

export function scrollToText(root: HTMLElement, texts: string[]) {
  const ranges = findAllRanges(root, texts);
  if (ranges.length === 0) return;

  // if the text appears more than once, pick the match closest to the middle of the screen
  const mid = window.innerHeight / 2;
  let best: { range: Range; rect: DOMRect } | null = null;
  let bestDist = Infinity;
  for (const range of ranges) {
    const rect = range.getClientRects()[0] ?? range.getBoundingClientRect();
    const dist = Math.abs(rect.top - mid);
    if (dist < bestDist) {
      bestDist = dist;
      best = { range, rect };
    }
  }
  if (!best) return;

  // put the start of the match about a third of the way down the scroll area
  const scroller = getScrollParent(best.range.startContainer.parentElement);
  if (scroller) {
    const top = scroller.getBoundingClientRect().top;
    scroller.scrollBy({
      top: best.rect.top - (top + scroller.clientHeight / 3),
      behavior: "smooth",
    });
  } else {
    window.scrollBy({ top: best.rect.top - window.innerHeight / 3, behavior: "smooth" });
  }
}