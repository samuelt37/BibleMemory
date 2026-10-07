import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import { useAuth } from "@clerk/clerk-react";
import { API_URL } from "@/constants/config";
import type { Note, NoteFilters } from "../types";

export function useNotesList(filters: NoteFilters = {}) {
  const { getToken, isSignedIn } = useAuth();
  const { q = "", book = null, chapter = null } = filters;

  return useQuery({
    queryKey: ["notes", "list", { q, book, chapter }],
    queryFn: async () => {
      const token = await getToken();
      const params = new URLSearchParams();
      if (q) params.set("q", q);
      if (book !== null) params.set("book", book);
      if (book !== null && chapter) params.set("chapter", chapter);
      const qs = params.toString();

      const res = await fetch(`${API_URL}/notes${qs ? `?${qs}` : ""}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Failed to fetch notes");
      return res.json() as Promise<Note[]>;
    },
    enabled: !!isSignedIn,
    placeholderData: keepPreviousData,
    refetchInterval: (query) => {
      const notes = query.state.data ?? [];
      const stillProcessing = notes.some((n) => n.status === "processing");
      return stillProcessing ? 3000 : false;
    },
  });
}

export function useUploadNote() {
  const { getToken } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (file: File) => {
      const token = await getToken();
      const formData = new FormData();
      formData.append("file", file);

      const res = await fetch(`${API_URL}/notes`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      });
      if (!res.ok) throw new Error("Upload failed");
      return res.json() as Promise<Note>;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
    },
  });
}

export function useUploadTextNote() {
  const { getToken } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ title, text }: { title: string; text: string }) => {
      const token = await getToken();
      const res = await fetch(`${API_URL}/notes/text`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ title, text }),
      });
      if (!res.ok) throw new Error("Failed to save note");
      return res.json() as Promise<Note>;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
    },
  });
}

export function useDeleteNote() {
  const { getToken } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (noteId: number) => {
      const token = await getToken();
      const res = await fetch(`${API_URL}/notes/${noteId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Failed to delete note");
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
    },
  });
}

export async function fetchNoteDownloadURL(getToken: () => Promise<string | null>, noteId: number): Promise<string> {
  const token = await getToken();
  const res = await fetch(`${API_URL}/notes/${noteId}/download`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error("Failed to get download link");
  const data = await res.json();
  return data.url;
}

export function useNote(noteId: number | null) {
  const { getToken, isSignedIn } = useAuth();

  return useQuery({
    queryKey: ["notes", noteId],
    queryFn: async () => {
      const token = await getToken();
      const res = await fetch(`${API_URL}/notes/${noteId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Failed to fetch note");
      return res.json();
    },
    enabled: noteId !== null && !!isSignedIn,
  });
}
