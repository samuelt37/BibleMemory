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

export type ScriptureRangeDTO = {
  startBookId: number;
  startChapter: number;
  startVerse: number | null;
  endBookId: number;
  endChapter: number;
  endVerse: number | null;
};

export function bookRangeToDTO(r: BookRange): ScriptureRangeDTO {
  return {
    startBookId: r.start.bookId,
    startChapter: r.start.chapter,
    startVerse: r.start.verse,
    endBookId: r.end.bookId,
    endChapter: r.end.chapter,
    endVerse: r.end.verse,
  };
}