import type { VerseRef, Chunk, DeletedRef } from "@/features/references";
import { apiFetch } from "@/lib/api";

export async function createChunk(
  noteId: number,
  content: string,
  refs: VerseRef[],
  token: string | null,
): Promise<Chunk> {
  const res = await apiFetch(`/notes/${noteId}/chunks`, token, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content, refs }),
  });
  if (!res.ok) throw new Error("failed to create chunk");
  return res.json();
}

export async function deleteRef(
  noteId: number,
  ref: VerseRef,
  token: string | null,
): Promise<DeletedRef[]> {
  const { bookId, chapter, verseStart, endBookId, endChapter, verseEnd } = ref;
  const res = await apiFetch(`/notes/${noteId}/refs`, token, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      bookId,
      chapter,
      verseStart,
      endBookId,
      endChapter,
      verseEnd,
    }),
  });
  if (!res.ok) throw new Error("failed to delete reference");
  return (await res.json()).deleted;
}

export async function fetchNoteCountsByBook(
  token: string | null,
): Promise<Record<number, number>> {
  const res = await apiFetch("/notes/counts-by-book", token);
  if (!res.ok) throw new Error("failed to load note counts");
  return res.json();
}