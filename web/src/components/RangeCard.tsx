// components/RangeCard.tsx
import { useState } from "react";
import { X } from "lucide-react";
import type { BookRange, ScripturePosition } from "@/models/BookRange";
import { isWholeChapterRange } from "@/models/BookRange";
import type { BookInfo } from "@/models/BookInfo";

type RangeCardProps = {
  range: BookRange;
  books: BookInfo[];
  onUpdate: (id: number, patch: Partial<BookRange>) => void;
  onRemove: (id: number) => void;
};

function parseChapterVerse(input: string): { chapter: number; verse: number | null } | null {
  const trimmed = input.trim();
  if (!trimmed) return null;

  const match = trimmed.match(/^(\d+)(?::(\d+))?$/);
  if (!match) return null;

  const chapter = Number(match[1]);
  const verse = match[2] ? Number(match[2]) : null;

  if (chapter < 1) return null;

  return { chapter, verse };
}

function comparePositions(a: ScripturePosition, b: ScripturePosition): number {
  if (a.bookId !== b.bookId) return a.bookId - b.bookId;
  if (a.chapter !== b.chapter) return a.chapter - b.chapter;
  return (a.verse ?? 0) - (b.verse ?? 0);
}

function PositionPicker({
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
    position.verse !== null ? `${position.chapter}:${position.verse}` : `${position.chapter}`
  );

  const commit = () => {
    const parsed = parseChapterVerse(text);
    if (!parsed) {
      setText(position.verse !== null ? `${position.chapter}:${position.verse}` : `${position.chapter}`);
      return;
    }
    const chapter = Math.min(parsed.chapter, maxChapters);
    onChange({ ...position, chapter, verse: parsed.verse });
    setText(parsed.verse !== null ? `${chapter}:${parsed.verse}` : `${chapter}`);
  };

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs text-muted-foreground">{label}</span>
      <div className="flex items-center gap-1">
        <select
          value={position.bookId}
          onChange={(e) => {
            const selected = books.find((b) => b.id === Number(e.target.value));
            if (!selected) return;
            onChange({ book: selected.book, bookId: selected.id, chapter: 1, verse: null });
            setText("1");
          }}
          className="rounded-md border p-1 text-sm bg-transparent"
        >
          {books.map((b) => (
            <option key={b.id} value={b.id}>{b.book}</option>
          ))}
        </select>
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

export function RangeCard({ range, books, onUpdate, onRemove }: RangeCardProps) {
  const [expanded, setExpanded] = useState(true);

  const startBook = books.find((b) => b.id === range.start.bookId);
  const isFullBook =
    isWholeChapterRange(range) &&
    range.start.chapter === 1 &&
    range.end.chapter === (startBook?.chapters ?? 1);

  const title = isFullBook
    ? range.start.book
    : isWholeChapterRange(range)
      ? range.start.chapter === range.end.chapter
        ? `${range.start.book} ${range.start.chapter}`
        : `${range.start.book} ${range.start.chapter}\u2013${range.end.chapter}`
      : range.start.bookId === range.end.bookId
        ? `${range.start.book} ${range.start.chapter}${range.start.verse !== null ? `:${range.start.verse}` : ""}\u2013${range.end.chapter}${range.end.verse !== null ? `:${range.end.verse}` : ""}`
        : `${range.start.book} ${range.start.chapter}${range.start.verse !== null ? `:${range.start.verse}` : ""} \u2013 ${range.end.book} ${range.end.chapter}${range.end.verse !== null ? `:${range.end.verse}` : ""}`;

  const handleStartChange = (pos: ScripturePosition) => {
    const patch: Partial<BookRange> = { start: pos };
    if (comparePositions(pos, range.end) > 0) {
    patch.end = { ...pos };
    }
    onUpdate(range.id, patch);
  };

  const handleEndChange = (pos: ScripturePosition) => {
    if (comparePositions(pos, range.start) < 0) return;
    onUpdate(range.id, { end: pos });
  };

  if (!expanded) {
    return (
      <button
        type="button"
        onClick={() => setExpanded(true)}
        className="flex items-center gap-2 h-10 px-4 border border-border rounded-lg bg-card text-card-foreground shadow-xs hover:bg-accent/5">
        <span className="italic">{title}</span>
        <span
          onClick={(e) => {
            e.stopPropagation();
            onRemove(range.id);
          }}
          className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
          aria-label="Remove range">
          <X className="size-4" />
        </span>
      </button>
    );
  }

  return (
    <div className="p-4 border rounded-lg bg-card text-card-foreground shadow-xs flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="italic text-sm">{title}</span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => setExpanded(false)}
            className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground text-xs">
            Done
          </button>
          <button
            type="button"
            onClick={() => onRemove(range.id)}
            className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
            aria-label="Remove range">
            <X className="size-4" />
          </button>
        </div>
      </div>

      <div className="flex items-center gap-3 flex-wrap">
        <PositionPicker
          label="From"
          position={range.start}
          books={books}
          onChange={handleStartChange}
        />
        <span className="text-muted-foreground mt-4">&ndash;</span>
        <PositionPicker
          label="To"
          position={range.end}
          books={books}
          onChange={handleEndChange}
        />
      </div>

      <p className="text-xs text-muted-foreground">Leave verse blank for whole chapter.</p>
    </div>
  );
}