import type { NoteRef } from "../references/types";

export type BookInfo = {
  id: number;
  book: string;
  chapters: number;
};

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

export type ReviewSession = {
  id: number;
  ranges: ScriptureRangeDTO[];
  bookmarked: boolean;
  createdAt: string;
};

export type ReviewResult = {
  accuracy: number;
  feedback: string;
  notes?: NoteRef[];
};

export type ReviewUnit = { key: string; label: string; range: BookRange };

export type ChapterResult = { accuracy: number; feedback: string };