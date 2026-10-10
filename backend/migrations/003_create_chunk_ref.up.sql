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

-- carry over the tags that exist today
INSERT INTO chunk_refs (user_id, note_id, chunk_id, book_id, chapter, verse_start, verse_end)
SELECT user_id, note_id, id, book_id, chapter, verse_start, verse_end
FROM note_chunks WHERE book_id IS NOT NULL;

ALTER TABLE note_chunks
    DROP COLUMN book_id,
    DROP COLUMN chapter,
    DROP COLUMN verse_start,
    DROP COLUMN verse_end;