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

function isVideo(file: File): boolean {
  if (file.type && file.type.startsWith("video/")) return true;
  const ext = file.name.split(".").pop()?.toLowerCase();
  return ["mp4", "mov", "mkv", "webm", "avi", "m4v", "wmv", "3gp"].includes(ext ?? "");
}

function getVideoContentType(file: File): string {
  if (file.type && file.type.startsWith("video/")) return file.type;
  const ext = file.name.split(".").pop()?.toLowerCase();
  const map: Record<string, string> = {
    mp4: "video/mp4",
    mov: "video/quicktime",
    m4v: "video/mp4",
    webm: "video/webm",
    mkv: "video/x-matroska",
    avi: "video/x-msvideo",
    wmv: "video/x-ms-wmv",
    "3gp": "video/3gpp",
  };
  return (ext && map[ext]) || "video/mp4";
}

export type UploadProgressCallback = (progress: {
  percent: number;
  stage: "requesting_url" | "uploading" | "completing";
  loadedBytes: number;
  totalBytes: number;
}) => void;

export type UploadNoteInput =
  | File
  | { file: File; onProgress?: UploadProgressCallback };

const PART_CONCURRENCY = 3;
const PART_RETRIES = 3;

function putPart(url: string, blob: Blob, onLoaded: (n: number) => void): Promise<string> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", url); // no Content-Type header; it was set when the upload was created
    xhr.upload.onprogress = (e) => onLoaded(e.loaded);
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        const etag = xhr.getResponseHeader("ETag");
        if (etag) resolve(etag);
        else reject(new Error("Missing ETag (R2 CORS must expose it)"));
      } else {
        reject(new Error(`Part upload failed (${xhr.status})`));
      }
    };
    xhr.onerror = () => reject(new Error("Part upload failed (network error)"));
    xhr.send(blob);
  });
}

async function uploadVideoNote(
  file: File,
  getToken: () => Promise<string | null>,
  onProgress?: UploadProgressCallback,
): Promise<Note> {
  const contentType = getVideoContentType(file);
  onProgress?.({ percent: 0, stage: "requesting_url", loadedBytes: 0, totalBytes: file.size });

  // 1. start the multipart upload
  const token1 = await getToken();
  const startRes = await fetch(`${API_URL}/notes/video/multipart/start`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token1}`, "Content-Type": "application/json" },
    body: JSON.stringify({ filename: file.name, contentType, size: file.size }),
  });
  if (!startRes.ok) throw new Error((await startRes.text()) || "Could not start upload");
  const { key, uploadId, partSize, urls } = (await startRes.json()) as {
    key: string;
    uploadId: string;
    partSize: number;
    urls: string[];
  };

  // 2. upload the parts, a few at a time, each with its own retries
  const loaded = new Array<number>(urls.length).fill(0);
  const etags = new Array<string>(urls.length);
  const report = () => {
    const sum = loaded.reduce((a, b) => a + b, 0);
    onProgress?.({
      percent: Math.min(99, Math.round((sum / file.size) * 100)),
      stage: "uploading",
      loadedBytes: sum,
      totalBytes: file.size,
    });
  };

  let next = 0;
  const worker = async () => {
    while (true) {
      const i = next++;
      if (i >= urls.length) return;
      const blob = file.slice(i * partSize, Math.min(file.size, (i + 1) * partSize));
      for (let attempt = 1; ; attempt++) {
        try {
          etags[i] = await putPart(urls[i], blob, (n) => {
            loaded[i] = n;
            report();
          });
          loaded[i] = blob.size;
          report();
          break;
        } catch (err) {
          loaded[i] = 0;
          if (attempt >= PART_RETRIES) throw err;
          await new Promise((r) => setTimeout(r, 1000 * attempt));
        }
      }
    }
  };
  await Promise.all(Array.from({ length: Math.min(PART_CONCURRENCY, urls.length) }, worker));

  // 3. finish (fresh token, since step 2 can outlast the old one)
  onProgress?.({ percent: 100, stage: "completing", loadedBytes: file.size, totalBytes: file.size });
  const token2 = await getToken();
  const doneRes = await fetch(`${API_URL}/notes/video/multipart/complete`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token2}`, "Content-Type": "application/json" },
    body: JSON.stringify({
      key,
      uploadId,
      filename: file.name,
      parts: etags.map((etag, i) => ({ partNumber: i + 1, etag })),
    }),
  });
  if (!doneRes.ok) throw new Error((await doneRes.text()) || "Could not finish upload");
  return (await doneRes.json()) as Note;
}

export function useUploadNote() {
  const { getToken } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: UploadNoteInput) => {
      const file = input instanceof File ? input : input.file;
      const onProgress = input instanceof File ? undefined : input.onProgress;

      // videos take the direct-to-R2 route
      if (isVideo(file)) {
        return uploadVideoNote(file, getToken, onProgress);
      }

      // everything else: multipart upload
      const token = await getToken();
      onProgress?.({ percent: 30, stage: "uploading", loadedBytes: 0, totalBytes: file.size });
      const formData = new FormData();
      formData.append("file", file);

      const res = await fetch(`${API_URL}/notes`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      });
      if (!res.ok) {
        const errorText = await res.text();
        console.error(`[Note Upload] Upload failed (${res.status}):`, errorText);
        throw new Error(errorText || "Upload failed");
      }
      onProgress?.({ percent: 100, stage: "completing", loadedBytes: file.size, totalBytes: file.size });
      const note = await res.json();
      console.log(`[Note Upload] Note #${note.id} uploaded successfully!`);
      return note as Note;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
    },
  });
}