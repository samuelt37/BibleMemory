// NotePill.tsx
import type { NoteRef } from "@/features/review";
import { noteColor } from "./noteColors";

export type NotePillProps = {
  note: NoteRef;
  onOpen?: (note: NoteRef) => void;
  onHover?: (noteId: number | null) => void;
  colorIndex?: number;
};

export function NotePill({
  note,
  onOpen,
  onHover,
  colorIndex = 0,
}: NotePillProps) {
  return (
    <button
      type="button"
      onClick={() => onOpen?.(note)}
      onMouseEnter={() => onHover?.(note.noteId)}
      onMouseLeave={() => onHover?.(null)}
      onFocus={() => onHover?.(note.noteId)}
      onBlur={() => onHover?.(null)}
      className={`rounded-full border px-2.5 py-0.5 text-xs ${noteColor(colorIndex).pill}`}
    >
      {note.title}
    </button>
  );
}
