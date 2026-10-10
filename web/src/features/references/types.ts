export type NoteRef = { 
  noteId: number; 
  title: string; 
  quote?: string; 
  summaryQuote?: string;
  noteQuote?: string;
};

export type VerseRef = {
  id?: number;
  bookId: number;
  chapter: number | null;
  verseStart: number | null;
  verseEnd: number | null;
  source?: string;
};

export type Chunk = { id: number; refs: VerseRef[] };