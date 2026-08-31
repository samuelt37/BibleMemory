// components/RangeCard.tsx
import { X } from "lucide-react";
import type { BookRange } from "@/models/BookRange";

type RangeCardProps = {
  range: BookRange;
  books: string[];
  chapterCounts: Record<string, number>;
  onUpdate: (id: number, patch: Partial<BookRange>) => void;
  onRemove: (id: number) => void;
};

export function RangeCard({ range, books, chapterCounts, onUpdate, onRemove }: RangeCardProps) {
  const maxChapters = chapterCounts[range.book] ?? 1;

  const clamp = (val: number) => Math.min(Math.max(val, 1), maxChapters);

  return (
    <div className="p-4 border rounded-lg bg-card text-card-foreground shadow-xs">
      <div className="flex items-center justify-between">
        <select
          value={range.book}
          onChange={(e) => onUpdate(range.id, { book: e.target.value, from: 1, to: 1 })}
          className="italic bg-transparent focus:outline-none"
        >
          {books.map((b) => (
            <option key={b} value={b}>{b}</option>
          ))}
        </select>

        <button
          type="button"
          onClick={() => onRemove(range.id)}
          className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
          aria-label="Remove range"
        >
          <X className="size-4" />
        </button>
      </div>

      <div className="mt-2 flex items-center gap-2 text-sm">
        <label className="flex items-center gap-1">
          From
          <input
            type="number"
            min={1}
            max={maxChapters}
            value={range.from}
            onChange={(e) => onUpdate(range.id, { from: Number(e.target.value) })}
            onBlur={(e) => onUpdate(range.id, { from: clamp(Number(e.target.value)) })}
            className="w-16 rounded-md border p-1 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </label>
        <label className="flex items-center gap-1">
          To
          <input
            type="number"
            min={1}
            max={maxChapters}
            value={range.to}
            onChange={(e) => onUpdate(range.id, { to: Number(e.target.value) })}
            onBlur={(e) => onUpdate(range.id, { to: clamp(Number(e.target.value)) })}
            className="w-16 rounded-md border p-1 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </label>
        <span className="text-muted-foreground">of {maxChapters} chapters</span>
      </div>
    </div>
  );
}