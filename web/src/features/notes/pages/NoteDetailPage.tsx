import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ChevronLeft } from "lucide-react";
import { useAuth } from "@clerk/clerk-react";

import { NoteView } from "../components/NoteView";
import type { Chunk, VerseRef } from "@/features/references/types";
import { RefsBox } from "@/features/references/components/RefBox";
import { RetagButton } from "@/features/references/components/RetagButton";
import { API_URL } from "@/constants/config";
import {
  useTextSelection,
  type SelectionInfo,
} from "../hooks/useTextSelection";
import {
  HighlightToolbar,
  type HighlightColor,
  type ParsedRange,
} from "../components/HighlightToolbar";
import { useBooks } from "@/features/scripture/api/useBooks";
import { createChunk, deleteRef } from "../api/api";
import { refKey } from "@/features/references/lib/ref";

export function NoteDetailPage() {
  const { data: books = [] } = useBooks();
  const { id } = useParams();
  const navigate = useNavigate();
  const { getToken } = useAuth();

  const [refreshKey, setRefreshKey] = useState(0);

  const [chunks, setChunks] = useState<Chunk[]>([]);
  const [chunksLoading, setChunksLoading] = useState(true);

  const noteRef = useRef<HTMLDivElement>(null);
  const { selection, clear } = useTextSelection(noteRef);

  const loadChunks = useCallback(async () => {
    if (!id) return;
    setChunksLoading(true);
    try {
      const token = await getToken();
      const res = await fetch(`${API_URL}/notes/${id}/chunks`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) setChunks(await res.json());
    } finally {
      setChunksLoading(false);
    }
  }, [id, getToken]);

  useEffect(() => {
    loadChunks();
  }, [loadChunks]);

  async function handleRetagged() {
    await loadChunks();
    setRefreshKey((k) => k + 1);
  }

  const [pending, setPending] = useState<{
    selection: SelectionInfo;
    color: HighlightColor;
  } | null>(null);

  const active = pending?.selection ?? selection;

  const cancelPending = useCallback(() => setPending(null), []);

  function handlePick(color: HighlightColor) {
    if (!selection) return;
    setPending({ selection, color });
  }

  async function handleAdd(range: ParsedRange) {
    if (!pending || !id) return;
    const noteId = Number(id);
    if (Number.isNaN(noteId)) return;

    const { start, end } = range;
    const crossBook = start.bookId !== end.bookId;
    const crossChapter = start.chapter !== end.chapter;

    try {
      const token = await getToken();
      const chunk = await createChunk(
        noteId,
        pending.selection.text,
        [
          {
            bookId: start.bookId,
            chapter: start.chapter,
            verseStart: start.verse,
            endBookId: crossBook ? end.bookId : undefined,
            endChapter: crossChapter ? end.chapter : undefined,
            verseEnd: end.verse ?? start.verse,
          },
        ],
        token,
      );
      setChunks((prev) => [...prev, chunk]);
      setPending(null);
      clear();
    } catch {
      // keep `pending` so the user can retry; show an error toast here if you have one
    }
  }

  async function handleDeleteRef(ref: VerseRef) {
    const noteId = Number(id);
    if (!id || Number.isNaN(noteId)) return;

    try {
      const token = await getToken();
      await deleteRef(noteId, ref, token);
      const key = refKey(ref);
      setChunks((prev) =>
        prev.map((c) => ({
          ...c,
          refs: c.refs.filter((r) => refKey(r) !== key),
        })),
      );
    } catch {
      // show an error toast
    }
  }

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div className="flex items-start justify-between">
          <button
            type="button"
            onClick={() => navigate(`/notes${location.search}`)}
            className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground w-fit"
          >
            <ChevronLeft size={16} />
            Back to notes
          </button>

          {id && <RetagButton noteId={Number(id)} onDone={handleRetagged} />}
        </div>

        {id ? (
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start">
            <div className="min-w-0 flex-1">
              <div ref={noteRef} className="min-w-0 flex-1">
                <NoteView key={refreshKey} noteId={Number(id)} />
              </div>
              {active && (
                <HighlightToolbar
                  selection={active}
                  color={pending?.color ?? null}
                  books={books}
                  onPick={handlePick}
                  onSubmit={handleAdd}
                  onCancel={cancelPending}
                />
              )}
            </div>
            <RefsBox
              chunks={chunks}
              loading={chunksLoading}
              onDelete={handleDeleteRef}
            />
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">Note not found.</p>
        )}
      </div>
    </div>
  );
}
