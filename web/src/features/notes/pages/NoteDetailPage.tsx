import { useParams, useNavigate } from "react-router-dom";
import { ChevronLeft } from "lucide-react";
import { NoteView } from "../components/NoteView";

export function NoteDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();

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

        {id ? (
          <NoteView noteId={Number(id)} />
        ) : (
          <p className="text-muted-foreground text-sm">Note not found.</p>
        )}
      </div>
    </div>
  );
}
