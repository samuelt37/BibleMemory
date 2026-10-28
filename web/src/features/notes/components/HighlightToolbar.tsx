// features/notes/components/HighlightToolbar.tsx
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import type { SelectionInfo } from "../hooks/useTextSelection";
import {
  type BookRange,
  type ScripturePosition,
  type BookInfo,
  ScripturePositionPicker,
  comparePositions,
} from "@/features/scripture";

export const HIGHLIGHT_COLORS = [
  { name: "yellow", class: "bg-yellow-300" },
  { name: "green", class: "bg-green-300" },
  { name: "blue", class: "bg-blue-300" },
  { name: "pink", class: "bg-pink-300" },
  { name: "purple", class: "bg-purple-300" },
] as const;

export type HighlightColor = (typeof HIGHLIGHT_COLORS)[number]["name"];
export type ParsedRange = Pick<BookRange, "start" | "end">;

type Props = {
  selection: SelectionInfo;
  color: HighlightColor | null; // non-null = expanded (ref) mode
  books: BookInfo[];
  onPick: (color: HighlightColor) => void;
  onSubmit: (range: ParsedRange) => void;
  onCancel: () => void;
};

export function HighlightToolbar({
  selection,
  color,
  books,
  onPick,
  onSubmit,
  onCancel,
}: Props) {
  const { rect } = selection;
  const rootRef = useRef<HTMLDivElement>(null);

  const [start, setStart] = useState<ScripturePosition>(() => ({
    book: books[0]?.book ?? "",
    bookId: books[0]?.id ?? 0,
    chapter: 1,
    verse: null,
  }));
  const [end, setEnd] = useState<ScripturePosition | null>(null);

  // reset when going back to the color row
  useEffect(() => {
    if (!color) setEnd(null);
  }, [color]);

  // click outside = cancel
  useEffect(() => {
    if (!color) return;
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) onCancel();
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [color, onCancel]);

  const handleStartChange = (pos: ScripturePosition) => {
    setStart(pos);
    if (end && comparePositions(pos, end) > 0) setEnd({ ...pos });
  };

  const handleEndChange = (pos: ScripturePosition) => {
    if (comparePositions(pos, start) < 0) return;
    setEnd(pos);
  };

  const swatch = HIGHLIGHT_COLORS.find((c) => c.name === color);
  const expanded = color !== null;
  const showBelow = rect.top < 56 || expanded; // expanded is taller, so drop below

  return (
    <div
      ref={rootRef}
      // only block focus-stealing for the color row; the expanded form needs real focus
      onMouseDown={(e) => {
        if (!expanded) e.preventDefault();
      }}
      style={{
        position: "fixed",
        top: showBelow ? rect.bottom + 8 : rect.top - 44,
        left: rect.left + rect.width / 2,
        transform: "translateX(-50%)",
      }}
      className="z-50 rounded-lg border bg-popover p-2 shadow-md"
    >
      {!expanded ? (
        <div className="flex gap-1.5">
          {HIGHLIGHT_COLORS.map((c) => (
            <button
              key={c.name}
              type="button"
              aria-label={`Highlight ${c.name}`}
              onClick={() => onPick(c.name)}
              className={`h-6 w-6 rounded-full ${c.class} hover:scale-110 transition-transform`}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className={`h-4 w-4 rounded-full ${swatch?.class}`} />
            <button
              type="button"
              onClick={onCancel}
              aria-label="Cancel"
              className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
            >
              <X className="size-4" />
            </button>
          </div>

          <ScripturePositionPicker
            label={end ? "From" : "Ref"}
            position={start}
            books={books}
            onChange={handleStartChange}
          />

          {end ? (
            <div className="flex items-end gap-1">
              <ScripturePositionPicker
                label="To"
                position={end}
                books={books}
                onChange={handleEndChange}
              />
              <button
                type="button"
                onClick={() => setEnd(null)}
                aria-label="Remove end"
                className="mb-1 rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
              >
                <X className="size-4" />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setEnd({ ...start })}
              className="w-fit text-xs text-muted-foreground hover:text-foreground"
            >
              + End
            </button>
          )}

          <button
            type="button"
            onClick={() => onSubmit({ start, end: end ?? { ...start } })}
            className="rounded bg-primary px-2 py-1 text-xs text-primary-foreground"
          >
            Add
          </button>
        </div>
      )}
    </div>
  );
}
