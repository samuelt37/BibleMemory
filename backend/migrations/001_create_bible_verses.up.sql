CREATE TABLE bible_verses (
    id SERIAL PRIMARY KEY,
    translation TEXT NOT NULL,
    testament TEXT NOT NULL,
    book_order INT NOT NULL,
    book TEXT NOT NULL,
    chapter INT NOT NULL,
    verse INT NOT NULL,
    text TEXT NOT NULL,

    UNIQUE(translation, book, chapter, verse)
);

CREATE INDEX idx_bible_lookup
ON bible_verses(translation, book, chapter);

CREATE INDEX idx_bible_book_order
ON bible_verses (translation, book_order);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    clerk_user_id TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notes (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'file',
    r2_key TEXT,
    mime_type TEXT,
    file_size BIGINT,
    raw_text TEXT,
    status TEXT NOT NULL DEFAULT 'processing',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS note_chunks (
  id SERIAL PRIMARY KEY,
  note_id INT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  embedding VECTOR(768),
  book_id INT,
  chapter INT,
  verse_start INT,
  verse_end INT,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_note_chunks_lookup ON note_chunks (user_id, book_id, chapter);

CREATE TABLE chunk_refs (
  id          SERIAL PRIMARY KEY,
  user_id     INT  NOT NULL,
  note_id     INT  NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  chunk_id    INT  NOT NULL REFERENCES note_chunks(id) ON DELETE CASCADE,
  book_id     INT  NOT NULL,
  chapter     INT,
  verse_start INT,
  verse_end   INT,
  source      TEXT NOT NULL DEFAULT 'auto',  -- 'auto' (tagger) or 'user'
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE NULLS NOT DISTINCT (chunk_id, book_id, chapter, verse_start, verse_end)
);
CREATE INDEX ON chunk_refs (chunk_id);
CREATE INDEX ON chunk_refs (book_id, chapter);
CREATE INDEX ON chunk_refs (user_id, note_id);

ALTER TABLE note_chunks ADD COLUMN source TEXT NOT NULL DEFAULT 'auto'; -- 'user' for highlight-created chunks

INSERT INTO chunk_refs (user_id, note_id, chunk_id, book_id, chapter, verse_start, verse_end)
SELECT user_id, note_id, id, book_id, chapter, verse_start, verse_end
FROM note_chunks WHERE book_id IS NOT NULL;

ALTER TABLE note_chunks
    DROP COLUMN book_id,
    DROP COLUMN chapter,
    DROP COLUMN verse_start,
    DROP COLUMN verse_end;

    ALTER TABLE chunk_refs
  ADD COLUMN end_book_id INT,
  ADD COLUMN end_chapter INT;

ALTER TABLE chunk_refs
  ADD CONSTRAINT chunk_refs_end_after_start CHECK (
    end_book_id IS NULL OR end_book_id >= book_id
  );

ALTER TABLE chunk_refs
  DROP CONSTRAINT chunk_refs_chunk_id_book_id_chapter_verse_start_verse_end_key;

ALTER TABLE chunk_refs
  ADD CONSTRAINT chunk_refs_unique UNIQUE NULLS NOT DISTINCT
    (chunk_id, book_id, chapter, verse_start, end_book_id, end_chapter, verse_end);