import { useState, useRef, useEffect } from "react";
import {
  Upload,
  FileText,
  X,
  Download,
  Type,
  Loader2,
  AlertCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  useNotesList,
  useUploadNote,
  useUploadTextNote,
  useDeleteNote,
  fetchNoteDownloadURL,
} from "../api/useNotes";
import { useAuth } from "@clerk/clerk-react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useBooks, BookSelect } from "@/features/scripture";
import { useQuery } from "@tanstack/react-query";
import { fetchNoteCountsByBook } from "../api/api";
import { useUploads, startUpload, dismissUpload } from "../uploads/uploadStore";

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
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  const q = params.get("q") ?? "";
  const book = params.get("book");
  const chapter = params.get("chapter");
  const hasFilters = q !== "" || book !== null;
  const { data: books = [] } = useBooks();

  const { data: notesData, isLoading } = useNotesList({ q, book, chapter });
  const notes = notesData ?? [];
  const { mutateAsync: uploadNoteAsync } = useUploadNote();
  const { mutate: uploadTextNote } = useUploadTextNote();
  const { mutate: deleteNote } = useDeleteNote();
  const { getToken } = useAuth();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [pasteMode, setPasteMode] = useState(false);
  const [pasteTitle, setPasteTitle] = useState("");
  const [pasteText, setPasteText] = useState("");
  const [searchText, setSearchText] = useState(q);

  const activeUploads = useUploads();

  const { data: noteCounts } = useQuery({
    queryKey: ["note-counts-by-book"],
    queryFn: async () => fetchNoteCountsByBook(await getToken()),
  });

  // write the search text to the URL after a short pause
  useEffect(() => {
    if (searchText === q) return;
    const t = setTimeout(() => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (searchText) next.set("q", searchText);
          else next.delete("q");
          return next;
        },
        { replace: true },
      );
    }, 300);
    return () => clearTimeout(t);
  }, [searchText, q, setParams]);

  const setPassage = (nextBook: string, nextChapter: string) => {
    setParams((prev) => {
      const next = new URLSearchParams(prev);
      if (nextBook !== "") next.set("book", nextBook);
      else next.delete("book");
      if (nextBook !== "" && nextChapter !== "")
        next.set("chapter", nextChapter);
      else next.delete("chapter");
      return next;
    });
  };

  const clearFilters = () => {
    setSearchText("");
    setParams({});
  };

  const navigate = useNavigate();

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;

    Array.from(files).forEach((file) => {
      startUpload(file, (onProgress) => uploadNoteAsync({ file, onProgress }));
    });
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

  const isUploadingActive = activeUploads.some((u) => !u.error);

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
            <Button
              disabled={isUploadingActive}
              onClick={() => fileInputRef.current?.click()}
            >
              {isUploadingActive ? (
                <>
                  <Loader2 size={16} className="mr-2 animate-spin" />
                  Uploading…
                </>
              ) : (
                <>
                  <Upload size={16} className="mr-2" />
                  Upload note
                </>
              )}
            </Button>
          </div>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            accept="image/*,application/pdf,text/plain,video/*"
            className="hidden"
            onChange={handleFileSelect}
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <input
            type="text"
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            placeholder="Search notes…"
            className="rounded-md border px-3 py-1.5 text-sm"
          />
          <BookSelect
            books={books}
            value={book ? Number(book) : null}
            placeholder="All books"
            onChange={(b) => setPassage(b ? String(b.id) : "", "")}
            noteCounts={noteCounts}
            className="rounded-md border bg-background px-2 py-1.5 text-sm"
          />
          {book !== null && (
            <input
              type="number"
              min={1}
              value={chapter ?? ""}
              onChange={(e) => setPassage(book, e.target.value)}
              placeholder="Chapter"
              className="w-24 rounded-md border px-2 py-1.5 text-sm"
            />
          )}
          {hasFilters && (
            <button
              type="button"
              onClick={clearFilters}
              className="text-xs underline text-muted-foreground hover:text-foreground"
            >
              Clear
            </button>
          )}
        </div>

        {activeUploads.length > 0 && (
          <div className="flex flex-col gap-2">
            {activeUploads.map((u) => (
              <div
                key={u.id}
                className={`p-3.5 border rounded-lg ${
                  u.error
                    ? "border-destructive/40 bg-destructive/5 text-destructive"
                    : "border-primary/20 bg-card text-card-foreground shadow-xs"
                } flex flex-col gap-2`}
              >
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-2 min-w-0">
                    {u.error ? (
                      <AlertCircle
                        size={18}
                        className="text-destructive shrink-0"
                      />
                    ) : (
                      <Loader2
                        size={18}
                        className="text-primary animate-spin shrink-0"
                      />
                    )}
                    <p className="text-sm font-medium truncate">{u.filename}</p>
                    <span className="text-xs text-muted-foreground shrink-0">
                      ({formatFileSize(u.fileSize)})
                    </span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span className="text-xs font-semibold">
                      {u.error ? "Failed" : `${u.percent}%`}
                    </span>
                    {u.error && (
                      <button
                        type="button"
                        onClick={() => dismissUpload(u.id)}
                        className="rounded p-1 text-muted-foreground hover:bg-accent/10 hover:text-foreground"
                        aria-label="Dismiss"
                      >
                        <X className="size-4" />
                      </button>
                    )}
                  </div>
                </div>

                {u.error ? (
                  <p className="text-xs text-destructive">{u.error}</p>
                ) : (
                  <>
                    <div className="w-full bg-muted rounded-full h-1.5 overflow-hidden">
                      <div
                        className="bg-primary h-full transition-all duration-200"
                        style={{ width: `${Math.max(5, u.percent)}%` }}
                      />
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {u.stage === "requesting_url" && "Preparing upload..."}
                      {u.stage === "uploading" &&
                        `Uploading to storage (${u.percent}%)...`}
                      {u.stage === "completing" &&
                        "Finalizing and starting note processing..."}
                    </p>
                  </>
                )}
              </div>
            ))}
          </div>
        )}

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
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setPasteMode(false)}
              >
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
            {hasFilters
              ? "No notes match these filters."
              : "No notes yet — upload a file or paste text and we'll match it to relevant verses."}
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            {notes.map((note) => (
              <div
                key={note.id}
                className="p-4 border rounded-lg bg-card text-card-foreground shadow-xs flex items-center gap-3"
              >
                <FileText
                  size={20}
                  className="text-muted-foreground shrink-0"
                />
                <div
                  className="flex-1 min-w-0 cursor-pointer"
                  onClick={() =>
                    navigate(`/notes/${note.id}${location.search}`)
                  }
                >
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
