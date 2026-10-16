import { useState } from "react";
import { X } from "lucide-react";
import type { BookRange, ScripturePosition, BookInfo } from "../types";
import { ScripturePositionPicker } from "./ScripturePositionPicker";
import { comparePositions, isWholeChapterRange } from "../lib/positions";

type RangeCardProps = {
  range: BookRange;
  books: BookInfo[];
  onUpdate: (id: number, patch: Partial<BookRange>) => void;
  onRemove: (id: number) => void;
};

export function RangeCard({
  range,
  books,
  onUpdate,
  onRemove,
}: RangeCardProps) {
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
        className="flex items-center gap-2 h-10 px-4 border border-border rounded-lg bg-card text-card-foreground shadow-xs hover:bg-accent/5"
      >
        <span className="italic">{title}</span>
        <span
          onClick={(e) => {
            e.stopPropagation();
            onRemove(range.id);
          }}
          className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
          aria-label="Remove range"
        >
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
            className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground text-xs"
          >
            Done
          </button>
          <button
            type="button"
            onClick={() => onRemove(range.id)}
            className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
            aria-label="Remove range"
          >
            <X className="size-4" />
          </button>
        </div>
      </div>

      <div className="flex items-center gap-3 flex-wrap">
        <ScripturePositionPicker
          label="From"
          position={range.start}
          books={books}
          onChange={handleStartChange}
        />
        <span className="text-muted-foreground mt-4">&ndash;</span>
        <ScripturePositionPicker
          label="To"
          position={range.end}
          books={books}
          onChange={handleEndChange}
        />
      </div>

      <p className="text-xs text-muted-foreground">
        Leave verse blank for whole chapter.
      </p>
    </div>
  );
}
