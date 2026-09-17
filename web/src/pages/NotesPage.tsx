// pages/NotesPage.tsx
import { useState, useRef } from "react";
import { Upload, FileText, X, Pencil, Check } from "lucide-react";
import { Button } from "@/components/ui/button";

type SuggestedMatch = {
  id: number;
  book: string;
  chapter: number;
  verse: number | null;
};

type Note = {
  id: number;
  name: string;
  uploadedAt: string;
  status: "matching" | "matched";
  matches: SuggestedMatch[];
};

const mockNotes: Note[] = [
  {
    id: 1,
    name: "Genesis 1 study notes.pdf",
    uploadedAt: "2026-09-14",
    status: "matched",
    matches: [{ id: 1, book: "Genesis", chapter: 1, verse: null }],
  },
  {
    id: 2,
    name: "James chapter outline.docx",
    uploadedAt: "2026-09-10",
    status: "matched",
    matches: [{ id: 1, book: "James", chapter: 1, verse: null }],
  },
];

function matchLabel(m: SuggestedMatch): string {
  return m.verse !== null ? `${m.book} ${m.chapter}:${m.verse}` : `${m.book} ${m.chapter}`;
}

export function NotesPage() {
  const [notes, setNotes] = useState<Note[]>(mockNotes);
  const [editingNoteId, setEditingNoteId] = useState<number | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;
    // TODO: upload to backend, backend runs matching, returns suggested matches
    console.log("Uploading:", files);
    e.target.value = "";
  };

  const removeMatch = (noteId: number, matchId: number) => {
    setNotes((prev) =>
      prev.map((n) =>
        n.id === noteId ? { ...n, matches: n.matches.filter((m) => m.id !== matchId) } : n
      )
    );
    // TODO: PATCH backend
  };

  const removeNote = (noteId: number) => {
    setNotes((prev) => prev.filter((n) => n.id !== noteId));
    // TODO: DELETE backend
  };

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold">Notes</h1>
          <Button onClick={() => fileInputRef.current?.click()}>
            <Upload size={16} className="mr-2" />
            Upload note
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            accept=".pdf,.doc,.docx,image/*"
            className="hidden"
            onChange={handleFileSelect}
          />
        </div>

        {notes.length === 0 ? (
          <p className="text-muted-foreground text-sm">
            No notes yet — upload a PDF, doc, or image and we'll match it to relevant verses.
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            {notes.map((note) => (
              <div key={note.id} className="p-4 border rounded-lg bg-card text-card-foreground shadow-xs flex flex-col gap-3">
                <div className="flex items-center gap-3">
                  <FileText size={20} className="text-muted-foreground shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm truncate">{note.name}</p>
                    <p className="text-xs text-muted-foreground">{note.uploadedAt}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeNote(note.id)}
                    className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
                    aria-label="Remove note"
                  >
                    <X className="size-4" />
                  </button>
                </div>

                {note.status === "matching" ? (
                  <p className="text-xs text-muted-foreground">Matching to verses…</p>
                ) : (
                  <div className="flex flex-wrap items-center gap-2">
                    {note.matches.map((m) => (
                      <span
                        key={m.id}
                        className="flex items-center gap-1.5 rounded-full bg-accent/10 text-accent-foreground pl-3 pr-1.5 py-1 text-sm"
                      >
                        {matchLabel(m)}
                        {editingNoteId === note.id && (
                          <button
                            type="button"
                            onClick={() => removeMatch(note.id, m.id)}
                            className="rounded-full p-0.5 hover:bg-accent/20"
                            aria-label={`Remove ${matchLabel(m)}`}
                          >
                            <X className="size-3" />
                          </button>
                        )}
                      </span>
                    ))}

                    <button
                      type="button"
                      onClick={() => setEditingNoteId(editingNoteId === note.id ? null : note.id)}
                      className="flex items-center gap-1 rounded-full border border-dashed px-3 py-1 text-xs text-muted-foreground hover:text-foreground hover:border-foreground"
                    >
                      {editingNoteId === note.id ? (
                        <>
                          <Check size={12} /> Done
                        </>
                      ) : (
                        <>
                          <Pencil size={12} /> Edit matches
                        </>
                      )}
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}