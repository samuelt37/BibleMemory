// NoteView.tsx
import { useNote } from "../api/useNotes";

export function NoteView({ noteId }: { noteId: number }) {
  const { data: note, isLoading } = useNote(noteId);

  if (isLoading)
    return <p className="text-muted-foreground text-sm">Loading…</p>;
  if (!note)
    return <p className="text-muted-foreground text-sm">Note not found.</p>;

  return (
    <>
      <h1 className="text-lg font-semibold">{note.filename}</h1>
      <p className="text-sm whitespace-pre-wrap text-card-foreground">
        {note.rawText ?? "No text available."}
      </p>
    </>
  );
}
