import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ChevronLeft } from "lucide-react";
import { useAuth } from "@clerk/clerk-react";

import { NoteView } from "../components/NoteView";
import { fetchChunks, type Chunk, type VerseRef } from "@/features/references";
import { RefsBox } from "@/features/references/components/RefBox";
import { RetagButton } from "@/features/references/components/RetagButton";
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
import { createChunk, deleteRef, fetchNoteMedia } from "../api/api";
import { refKey } from "@/features/references/lib/ref";
import { useRangeHighlight, scrollToText } from "../hooks/useRangeHighlight";
import { NoteMediaViewer } from "../components/NoteMediaViewer";
import type { NoteMedia } from "../types";

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
    const noteId = Number(id);
    if (!id || Number.isNaN(noteId)) return;
    setChunksLoading(true);
    try {
      const token = await getToken();
      const data = await fetchChunks(noteId, token);
      setChunks(data);
    } catch (err) {
      console.error("Failed to load chunks:", err);
    } finally {
      setChunksLoading(false);
    }
  }, [id, getToken]);

  useEffect(() => {
    loadChunks();
  }, [loadChunks]);

  const [media, setMedia] = useState<NoteMedia | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const noteId = Number(id);
    if (!id || Number.isNaN(noteId)) return;
    let cancelled = false;
    (async () => {
      try {
        const token = await getToken();
        const m = await fetchNoteMedia(noteId, token);
        if (!cancelled) setMedia(m);
      } catch (err) {
        console.error("Failed to load media:", err);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id, getToken]);

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

  const [hoveredRef, setHoveredRef] = useState<VerseRef | null>(null);
  const [pinnedRef, setPinnedRef] = useState<VerseRef | null>(null);

  // chunks that contain this ref (whole-book pill = any ref in that book)
  const chunksFor = useCallback(
    (ref: VerseRef | null) => {
      if (!ref) return [];
      if (ref.chapter == null) {
        return chunks.filter((c) =>
          c.refs?.some((r) => r.bookId === ref.bookId),
        );
      }
      const key = refKey(ref);
      return chunks.filter((c) => c.refs?.some((r) => refKey(r) === key));
    },
    [chunks],
  );

  const textsFor = useCallback(
    (ref: VerseRef | null) => {
      if (!ref) return [];
      if (ref.chapter == null) return chunksFor(ref).map((c) => c.content);
      const key = refKey(ref);
      return chunksFor(ref).flatMap((c) => {
        const match = c.refs?.find((r) => refKey(r) === key);
        return match?.quotes?.length ? match.quotes : [c.content];
      });
    },
    [chunksFor],
  );

  // hover previews on top of the pin; leaving a pill falls back to the pinned ref
  const shownTexts = useMemo(
    () => textsFor(hoveredRef ?? pinnedRef),
    [textsFor, hoveredRef, pinnedRef],
  );

  useRangeHighlight(noteRef, shownTexts);

  function seekTo(seconds: number) {
    const v = videoRef.current;
    if (!v) return;
    v.currentTime = Math.max(0, seconds - 2); // small lead-in
    v.play().catch(() => {}); // browsers may block autoplay; the seek still happens
  }

  function handleSelectRef(ref: VerseRef) {
    if (pinnedRef && refKey(pinnedRef) === refKey(ref)) {
      setPinnedRef(null); // click again to unpin
      return;
    }
    setPinnedRef(ref);
    if (noteRef.current) scrollToText(noteRef.current, textsFor(ref));

    // jump the video to the earliest chunk that mentions this ref
    const times = chunksFor(ref)
      .map((c) => c.startSeconds)
      .filter((s): s is number => s != null);
    if (times.length) seekTo(Math.min(...times));
  }

  // Escape unpins
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setPinnedRef(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

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
    setHoveredRef(null);
    if (pinnedRef && refKey(pinnedRef) === refKey(ref)) setPinnedRef(null);
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
      {/* page scrolls normally below lg; on lg the columns scroll internally */}
      <div className="flex-1 min-h-0 overflow-y-auto lg:overflow-hidden p-6 flex flex-col gap-4">
        <div className="flex items-start justify-between shrink-0">
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
          <div className="flex flex-col gap-4 lg:flex-row lg:items-stretch lg:flex-1 lg:min-h-0">
            {/* left column: media stays put, text scrolls */}
            <div className="min-w-0 flex-1 flex flex-col lg:min-h-0">
              {media && (
                <div className="sticky top-0 z-10 mb-4 bg-background pb-2 lg:static lg:shrink-0">
                  <NoteMediaViewer ref={videoRef} media={media} />
                </div>
              )}

              <div className="lg:flex-1 lg:min-h-0 lg:overflow-y-auto">
                <div ref={noteRef} className="min-w-0 flex-1">
                  <NoteView key={refreshKey} noteId={Number(id)} />
                </div>
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

            {/* refs scroll on their own */}
            <div className="lg:min-h-0 lg:overflow-y-auto">
              <RefsBox
                chunks={chunks}
                loading={chunksLoading}
                onDelete={handleDeleteRef}
                onHoverRef={setHoveredRef}
                onSelectRef={handleSelectRef}
                pinnedKey={pinnedRef ? refKey(pinnedRef) : null}
              />
            </div>
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">Note not found.</p>
        )}
      </div>
    </div>
  );
}
