export type Note = {
  id: number;
  filename: string;
  sourceType: "file" | "text";
  mimeType?: string;
  fileSize?: number;
  status: "processing" | "ready" | "failed";
  createdAt: string;
  updatedAt: string;
};