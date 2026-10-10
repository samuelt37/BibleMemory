// RefPill.tsx
import { refColor } from "../lib/refColors";
import type { NoteRef } from "../types";

export type RefPillProps = {
  note: NoteRef;
  onOpen?: (note: NoteRef) => void;
  onHover?: (noteId: number | null) => void;
  colorIndex?: number;
};

export function RefPill({
  note,
  onOpen,
  onHover,
  colorIndex = 0,
}: RefPillProps) {
  return (
    <button
      type="button"
      onClick={() => onOpen?.(note)}
      onMouseEnter={() => onHover?.(note.noteId)}
      onMouseLeave={() => onHover?.(null)}
      onFocus={() => onHover?.(note.noteId)}
      onBlur={() => onHover?.(null)}
      className={`rounded-full border px-2.5 py-0.5 text-xs ${refColor(colorIndex).pill}`}
    >
      {note.title}
    </button>
  );
}
