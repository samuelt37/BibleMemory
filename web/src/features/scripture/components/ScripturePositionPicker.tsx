import { useState } from "react";
import type { ScripturePosition, BookInfo } from "../types"; // adjust path
import { parseChapterVerse } from "../lib/positions";
import { BookSelect } from "./BookSelect";

export function ScripturePositionPicker({
  label,
  position,
  books,
  onChange,
}: {
  label: string;
  position: ScripturePosition;
  books: BookInfo[];
  onChange: (pos: ScripturePosition) => void;
}) {
  const book = books.find((b) => b.id === position.bookId);
  const maxChapters = book?.chapters ?? 1;

  const [text, setText] = useState(
    position.verse !== null
      ? `${position.chapter}:${position.verse}`
      : `${position.chapter}`,
  );

  const commit = () => {
    const parsed = parseChapterVerse(text);
    if (!parsed) {
      setText(
        position.verse !== null
          ? `${position.chapter}:${position.verse}`
          : `${position.chapter}`,
      );
      return;
    }
    const chapter = Math.min(parsed.chapter, maxChapters);
    onChange({ ...position, chapter, verse: parsed.verse });
    setText(
      parsed.verse !== null ? `${chapter}:${parsed.verse}` : `${chapter}`,
    );
  };

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs text-muted-foreground">{label}</span>
      <div className="flex items-center gap-1">
        <BookSelect
          books={books}
          value={position.bookId}
          onChange={(selected) => {
            if (!selected) return;
            onChange({
              book: selected.book,
              bookId: selected.id,
              chapter: 1,
              verse: null,
            });
            setText("1");
          }}
          className="rounded-md border p-1 text-sm bg-transparent"
        />
        <input
          type="text"
          value={text}
          placeholder="1 or 1:10"
          onChange={(e) => setText(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === "Enter") (e.target as HTMLInputElement).blur();
          }}
          className="w-20 rounded-md border p-1 text-sm"
        />
      </div>
    </div>
  );
}
