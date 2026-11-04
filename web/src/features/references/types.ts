export type NoteRef = { 
  noteId: number; 
  title: string; 
  quote?: string; 
  summaryQuote?: string;
  noteQuote?: string;
};

export interface VerseRef {
  id?: number;
  bookId: number;
  chapter?: number | null;
  verseStart?: number | null;
  endBookId?: number | null;
  endChapter?: number | null;
  verseEnd?: number | null;
  source?: string;
  quotes?: string[];
}

export type Chunk = { id: number; content: string; refs: VerseRef[] };
export type DeletedRef = VerseRef & { chunkId: number };