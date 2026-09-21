// pages/NotesPage.tsx
import { useState, useRef } from "react";
import { Upload, FileText, X, Download, Type } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  useNotesList,
  useUploadNote,
  useUploadTextNote,
  useDeleteNote,
  fetchNoteDownloadURL,
} from "@/hooks/useNotes";
import { useAuth } from "@clerk/clerk-react";

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function statusLabel(status: string): string {
  switch (status) {
    case "processing":
      return "Processing…";
    case "ready":
      return "Ready";
    case "failed":
      return "Failed to process";
    default:
      return status;
  }
}

export function NotesPage() {
  const { data: notesData, isLoading } = useNotesList();
  const notes = notesData ?? [];
  const { mutate: uploadNote } = useUploadNote();
  const { mutate: uploadTextNote } = useUploadTextNote();
  const { mutate: deleteNote } = useDeleteNote();
  const { getToken } = useAuth();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [pasteMode, setPasteMode] = useState(false);
  const [pasteTitle, setPasteTitle] = useState("");
  const [pasteText, setPasteText] = useState("");

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;
    Array.from(files).forEach((file) => uploadNote(file));
    e.target.value = "";
  };

  const handlePasteSubmit = () => {
    if (!pasteText.trim()) return;
    uploadTextNote({ title: pasteTitle, text: pasteText });
    setPasteTitle("");
    setPasteText("");
    setPasteMode(false);
  };

  const handleDownload = async (noteId: number) => {
    const url = await fetchNoteDownloadURL(getToken, noteId);
    window.open(url, "_blank");
  };

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold">Notes</h1>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setPasteMode((v) => !v)}>
              <Type size={16} className="mr-2" />
              Paste text
            </Button>
            <Button onClick={() => fileInputRef.current?.click()}>
              <Upload size={16} className="mr-2" />
              Upload note
            </Button>
          </div>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            accept=".pdf,.doc,.docx,image/*"
            className="hidden"
            onChange={handleFileSelect}
          />
        </div>

        {pasteMode && (
          <div className="p-4 border rounded-lg bg-card flex flex-col gap-2">
            <input
              type="text"
              placeholder="Title (optional)"
              value={pasteTitle}
              onChange={(e) => setPasteTitle(e.target.value)}
              className="rounded-md border p-2 text-sm"
            />
            <textarea
              placeholder="Paste your notes here..."
              value={pasteText}
              onChange={(e) => setPasteText(e.target.value)}
              rows={6}
              className="rounded-md border p-2 text-sm resize-none"
            />
            <div className="flex justify-end gap-2">
              <Button variant="ghost" size="sm" onClick={() => setPasteMode(false)}>
                Cancel
              </Button>
              <Button size="sm" onClick={handlePasteSubmit}>
                Save
              </Button>
            </div>
          </div>
        )}

        {isLoading ? (
          <p className="text-muted-foreground text-sm">Loading…</p>
        ) : notes.length === 0 ? (
          <p className="text-muted-foreground text-sm">
            No notes yet — upload a file or paste text and we'll match it to relevant verses.
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            {notes.map((note) => (
              <div key={note.id} className="p-4 border rounded-lg bg-card text-card-foreground shadow-xs flex items-center gap-3">
                <FileText size={20} className="text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{note.filename}</p>
                  <p className="text-xs text-muted-foreground">
                    {note.sourceType === "file" && note.fileSize
                      ? `${formatFileSize(note.fileSize)} · `
                      : ""}
                    {statusLabel(note.status)}
                  </p>
                </div>
                {note.sourceType === "file" && (
                  <button
                    type="button"
                    onClick={() => handleDownload(note.id)}
                    className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
                    aria-label="Download"
                  >
                    <Download className="size-4" />
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => deleteNote(note.id)}
                  className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
                  aria-label="Remove note"
                >
                  <X className="size-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}