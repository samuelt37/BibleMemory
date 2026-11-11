import { useSyncExternalStore } from "react";

export type UploadStage = "requesting_url" | "uploading" | "completing";

export interface ActiveUpload {
  id: string;
  filename: string;
  fileSize: number;
  percent: number;
  stage: UploadStage;
  error?: string;
}

let uploads: ActiveUpload[] = [];
const listeners = new Set<() => void>();
const emit = () => listeners.forEach((l) => l());

function subscribe(l: () => void) {
  listeners.add(l);
  return () => listeners.delete(l);
}

export function useUploads() {
  return useSyncExternalStore(subscribe, () => uploads);
}

function patch(id: string, p: Partial<ActiveUpload>) {
  uploads = uploads.map((u) => (u.id === id ? { ...u, ...p } : u));
  emit();
}

export function dismissUpload(id: string) {
  uploads = uploads.filter((u) => u.id !== id);
  emit();
}

export function startUpload(
  file: File,
  run: (onProgress: (p: { percent: number; stage: UploadStage }) => void) => Promise<unknown>,
) {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
  uploads = [
    ...uploads,
    { id, filename: file.name, fileSize: file.size, percent: 0, stage: "requesting_url" },
  ];
  emit();

  run(({ percent, stage }) => patch(id, { percent, stage }))
    .then(() => dismissUpload(id))
    .catch((err: any) => {
      console.error(`Upload failed for ${file.name}:`, err);
      patch(id, { error: err?.message || "Upload failed" });
    });
}

// Refresh or close mid-upload kills it, so warn (works from any page)
if (typeof window !== "undefined") {
  window.addEventListener("beforeunload", (e) => {
    if (uploads.some((u) => !u.error)) e.preventDefault();
  });
}