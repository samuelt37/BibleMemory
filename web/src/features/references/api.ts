// features/references/api.ts
import { apiFetch } from "@/lib/api";
import type { Chunk } from "./types";

export const rereferenceNote = (noteId: number, token: string | null) =>
  apiFetch(`/notes/${noteId}/rereference`, token, { method: "POST" });

export async function fetchChunks(noteId: number, token: string | null): Promise<Chunk[]> {
  const res = await apiFetch(`/notes/${noteId}/chunks`, token);
  return res.json();
}