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

export function dtoToBookRange(
  dto: ScriptureRangeDTO,
  books: BookInfo[],
  id: number = 1
): BookRange {
  const startBook = books.find((b) => b.id === dto.startBookId);
  const endBook = books.find((b) => b.id === dto.endBookId);
  return {
    id,
    start: {
      book: startBook?.book ?? "",
      bookId: dto.startBookId,
      chapter: dto.startChapter,
      verse: dto.startVerse,
    },
    end: {
      book: endBook?.book ?? "",
      bookId: dto.endBookId,
      chapter: dto.endChapter,
      verse: dto.endVerse,
    },
  };
}
