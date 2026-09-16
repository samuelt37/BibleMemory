// models/Session.ts
import type { ScriptureRangeDTO } from "./BookRange";

export type MemorySession = {
  id: number;
  ranges: ScriptureRangeDTO[];
  bookmarked: boolean;
  createdAt: string;
};