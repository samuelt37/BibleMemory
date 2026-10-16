import type { NoteRef } from "../references/types";
import type { BookRange, ScriptureRangeDTO } from "@/features/scripture";

export type ReviewSession = {
  id: number;
  ranges: ScriptureRangeDTO[];
  bookmarked: boolean;
  createdAt: string;
};

export type ReviewResult = {
  accuracy: number;
  feedback: string;
  notes?: NoteRef[];
};

export type ReviewUnit = { key: string; label: string; range: BookRange };

export type ChapterResult = { accuracy: number; feedback: string };