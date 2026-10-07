// NoteView.tsx
import { useNote } from "../api/useNotes";
import { useEffect, useRef } from "react";

function findQuote(text: string, quote: string): [number, number] | null {
  const words = quote
    .trim()
    .split(/\s+/)
    .map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  if (words.length === 0 || words[0] === "") return null;
  const m = new RegExp(words.join("\\s+"), "i").exec(text);
  return m ? [m.index, m.index + m[0].length] : null;
}

export function NoteView({
  noteId,
  highlight,
  markClass,
}: {
  noteId: number;
  highlight?: string;
  markClass?: string;
}) {
  const { data: note, isLoading } = useNote(noteId);
  const markRef = useRef<HTMLElement>(null);

  useEffect(() => {
    markRef.current?.scrollIntoView({ block: "center", behavior: "smooth" });
  }, [note, highlight]);

  if (isLoading)
    return <p className="text-muted-foreground text-sm">Loading…</p>;
  if (!note)
    return <p className="text-muted-foreground text-sm">Note not found.</p>;

  const text: string = note.rawText ?? "No text available.";
  const hit = highlight ? findQuote(text, highlight) : null;

  return (
    <>
      <h1 className="text-lg font-semibold">{note.filename}</h1>
      <p className="text-sm whitespace-pre-wrap text-card-foreground">
        {hit ? (
          <>
            {text.slice(0, hit[0])}
            <mark
              ref={markRef}
              className={
                markClass ??
                "rounded bg-yellow-200/70 px-0.5 text-inherit dark:bg-yellow-500/30"
              }
            >
              {text.slice(hit[0], hit[1])}
            </mark>
            {text.slice(hit[1])}
          </>
        ) : (
          text
        )}
      </p>
    </>
  );
}
