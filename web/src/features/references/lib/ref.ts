import type { VerseRef } from "../types";

export function refKey(r: VerseRef): string {
  return `${r.bookId}:${r.chapter}:${r.verseStart}:${r.verseEnd}`;
}

export function formatRef(r: VerseRef, bookName: string): string {
  if (r.chapter == null) return bookName;
  if (r.verseStart == null) return `${bookName} ${r.chapter}`;
  if (r.verseEnd == null || r.verseStart === r.verseEnd)
    return `${bookName} ${r.chapter}:${r.verseStart}`;
  return `${bookName} ${r.chapter}:${r.verseStart}-${r.verseEnd}`;
}