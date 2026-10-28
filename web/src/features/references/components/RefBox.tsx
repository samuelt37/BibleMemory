import { useBooks } from "@/features/scripture";
import { useMemo } from "react";
import { refKey, formatRef } from "../lib/ref";
import type { Chunk, VerseRef } from "../types";
import { RefPill } from "./RefPill";

type Props = {
  chunks: Chunk[];
  loading?: boolean;
  onDelete?: (ref: VerseRef) => void;
};

export function RefsBox({ chunks, loading, onDelete }: Props) {
  const { data: books = [], isPending, isError } = useBooks();

  // look up by id, not array index, in case ids don't match positions
  const bookNames = useMemo(
    () => new Map(books.map((b) => [b.id, b.book])),
    [books],
  );

  const refs = useMemo(() => {
    const seen = new Map<string, VerseRef>();
    for (const c of chunks) {
      for (const r of c.refs ?? []) {
        const key = refKey(r);
        if (!seen.has(key)) seen.set(key, r);
      }
    }
    return [...seen.values()].sort(
      (a, b) =>
        a.bookId - b.bookId ||
        (a.chapter ?? 0) - (b.chapter ?? 0) ||
        (a.verseStart ?? 0) - (b.verseStart ?? 0),
    );
  }, [chunks]);

  const showLoading = loading || isPending;

  return (
    <aside className="w-full lg:w-72 lg:shrink-0 lg:sticky lg:top-4 lg:self-start lg:max-h-[calc(100dvh-8rem)] flex flex-col rounded-lg border p-4">
      <h2 className="text-sm font-medium mb-3 shrink-0">
        References{!showLoading && refs.length > 0 && ` (${refs.length})`}
      </h2>
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
        {showLoading ? (
          <div className="flex flex-wrap gap-2" aria-busy="true">
            {[72, 96, 64, 88].map((w, i) => (
              <span
                key={i}
                className="h-6 rounded-full bg-muted animate-pulse"
                style={{ width: w }}
              />
            ))}
          </div>
        ) : isError ? (
          <p className="text-sm text-red-600">Couldn't load book names.</p>
        ) : refs.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No references found in this note.
          </p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {refs.map((r) => (
              <RefPill
                key={refKey(r)}
                label={formatRef(
                  r,
                  bookNames.get(r.bookId) ?? `Book ${r.bookId}`,
                )}
                title={
                  r.source === "user"
                    ? "Added by you"
                    : "Detected automatically"
                }
                className={
                  "font-medium " +
                  (r.source === "user"
                    ? "bg-primary/10 border-primary/30 text-primary"
                    : "bg-muted text-foreground")
                }
                onDelete={onDelete ? () => onDelete(r) : undefined}
              />
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}
