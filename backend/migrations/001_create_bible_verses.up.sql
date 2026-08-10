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
