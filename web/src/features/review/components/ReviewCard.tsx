import { useState } from "react";
import { noteColor } from "@/features/notes/components/noteColors";
import { NotePill } from "@/features/notes/components/NotePill";
import type { ReviewResult } from "../types";
import { HighlightedTextarea } from "@/features/notes/components/HighlightedTextarea";

type ReviewCardProps = {
  sectionTitle: string;
  value: string;
  onChange: (value: string) => void;
  result?: ReviewResult;
  onOpenNote?: (noteId: number) => void;
  onViewAllNotes?: () => void;
};

function renderFeedback(feedback: string, quote?: string, markClass?: string) {
  if (!quote) return feedback;
  const idx = feedback.indexOf(quote);
  if (idx === -1) return feedback;
  return (
    <>
      {feedback.slice(0, idx)}
      <mark className={`rounded px-0.5 text-inherit ${markClass ?? ""}`}>
        {quote}
      </mark>
      {feedback.slice(idx + quote.length)}
    </>
  );
}

export function ReviewCard({
  sectionTitle,
  value,
  onChange,
  result,
  onOpenNote,
  onViewAllNotes,
}: ReviewCardProps) {
  const [hoveredNoteId, setHoveredNoteId] = useState<number | null>(null);
  const hoveredIndex =
    result?.notes?.findIndex((n) => n.noteId === hoveredNoteId) ?? -1;
  const hovered = hoveredIndex >= 0 ? result?.notes?.[hoveredIndex] : undefined;

  return (
    <div className="p-4 border rounded-lg bg-card text-card-foreground shadow-xs">
      <p className="italic">{sectionTitle}</p>

      <HighlightedTextarea
        value={value}
        onChange={onChange}
        highlight={hovered?.summaryQuote}
        markClass={hovered ? noteColor(hoveredIndex).mark : undefined}
      />

      {result && (
        <div className="mt-2">
          <p
            className={`text-sm ${result.accuracy >= 7 ? "text-green-600" : result.accuracy >= 4 ? "text-yellow-600" : "text-red-600"}`}
          >
            {result.accuracy}/10 —{" "}
            {renderFeedback(
              result.feedback,
              hovered?.quote,
              hovered ? noteColor(hoveredIndex).mark : undefined,
            )}
          </p>

          {result.notes && result.notes.length > 0 && (
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <span className="text-xs text-muted-foreground">Ref:</span>
              {result.notes.map((n, i) => (
                <NotePill
                  key={n.noteId}
                  note={n}
                  onOpen={onOpenNote}
                  onHover={setHoveredNoteId}
                  colorIndex={i}
                />
              ))}
              <button
                type="button"
                onClick={onViewAllNotes}
                className="text-xs underline text-muted-foreground hover:text-foreground"
              >
                View all related notes
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
