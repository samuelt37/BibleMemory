import type { VerseRef } from "../types";

export function refKey(r: VerseRef): string {
  return `${r.bookId}:${r.chapter ?? ""}:${r.verseStart ?? ""}-${r.endBookId ?? ""}:${r.endChapter ?? ""}:${r.verseEnd ?? ""}`;
}

export function formatRef(r: VerseRef, bookName: string): string {
  if (r.chapter == null) return bookName;
  if (r.verseStart == null) return `${bookName} ${r.chapter}`;
  if (r.endChapter != null && r.endChapter !== r.chapter) {
    const startStr = r.verseStart != null ? `:${r.verseStart}` : "";
    const endStr = r.verseEnd != null ? `:${r.verseEnd}` : "";
    return `${bookName} ${r.chapter}${startStr}–${r.endChapter}${endStr}`;
  }
  if (r.verseEnd == null || r.verseStart === r.verseEnd)
    return `${bookName} ${r.chapter}:${r.verseStart}`;
  return `${bookName} ${r.chapter}:${r.verseStart}-${r.verseEnd}`;
}