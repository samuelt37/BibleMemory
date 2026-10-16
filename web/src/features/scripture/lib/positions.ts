import type { BookRange, ScripturePosition } from "../types";

export function isWholeChapterRange(r: BookRange): boolean {
  return (
    r.start.verse === null &&
    r.end.verse === null &&
    r.start.bookId === r.end.bookId
  );
}

export function parseChapterVerse(
  input: string,
): { chapter: number; verse: number | null } | null {
  const trimmed = input.trim();
  if (!trimmed) return null;

  const match = trimmed.match(/^(\d+)(?::(\d+))?$/);
  if (!match) return null;

  const chapter = Number(match[1]);
  const verse = match[2] ? Number(match[2]) : null;

  if (chapter < 1) return null;

  return { chapter, verse };
}

export function comparePositions(
  a: ScripturePosition,
  b: ScripturePosition,
): number {
  if (a.bookId !== b.bookId) return a.bookId - b.bookId;
  if (a.chapter !== b.chapter) return a.chapter - b.chapter;
  return (a.verse ?? 0) - (b.verse ?? 0);
}