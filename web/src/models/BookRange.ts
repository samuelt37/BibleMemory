// models/BookRange.ts

export type ScripturePosition = {
  book: string;
  bookId: number;
  chapter: number;
  verse: number | null; // null = whole chapter at this boundary
};

export type BookRange = {
  id: number;
  start: ScripturePosition;
  end: ScripturePosition;
};

export function isWholeChapterRange(r: BookRange): boolean {
  return (
    r.start.verse === null &&
    r.end.verse === null &&
    r.start.bookId === r.end.bookId
  );
}