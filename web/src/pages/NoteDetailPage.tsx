import { useParams, useNavigate } from "react-router-dom";
import { ChevronLeft } from "lucide-react";
import { useNote } from "@/hooks/useNotes";

export function NoteDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { data: note, isLoading } = useNote(id ? Number(id) : null);

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <button
          type="button"
          onClick={() => navigate("/notes")}
          className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground w-fit"
        >
          <ChevronLeft size={16} />
          Back to notes
        </button>

        {isLoading ? (
          <p className="text-muted-foreground text-sm">Loading…</p>
        ) : !note ? (
          <p className="text-muted-foreground text-sm">Note not found.</p>
        ) : (
          <>
            <h1 className="text-lg font-semibold">{note.filename}</h1>
            <p className="text-sm whitespace-pre-wrap text-card-foreground">
              {note.rawText ?? "No text available."}
            </p>
          </>
        )}
      </div>
    </div>
  );
}