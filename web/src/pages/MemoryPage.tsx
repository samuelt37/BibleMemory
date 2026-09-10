// pages/MemoryPage.tsx
import { useState } from "react";
import { MemoryCard } from "../components/MemoryCard.tsx";
import { RangeCard } from "../components/RangeCard.tsx";
import { useBooks } from "@/hooks/Books.ts";
import { Button } from "@/components/ui/button";
import { API_URL } from "@/constants/config";
import type { BookRange } from "@/models/BookRange";
import { isWholeChapterRange } from "@/models/BookRange";
import type { BookInfo } from "@/models/BookInfo.ts";

type MemoryUnit =
  | { kind: "chapter"; key: string; book: string; bookId: number; chapter: number; label: string }
  | { kind: "range"; key: string; label: string; range: BookRange };

type ChapterResult = { accuracy: number; feedback: string };

function buildMemoryUnits(ranges: BookRange[]): MemoryUnit[] {
  const units: MemoryUnit[] = [];
  const seen = new Set<string>();

  for (const r of ranges) {
    if (isWholeChapterRange(r)) {
      for (let ch = r.start.chapter; ch <= r.end.chapter; ch++) {
        const key = `${r.start.bookId}-${ch}`;
        if (seen.has(key)) continue;
        seen.add(key);
        units.push({
          kind: "chapter",
          key,
          book: r.start.book,
          bookId: r.start.bookId,
          chapter: ch,
          label: `${r.start.book} — Chapter ${ch}`,
        });
      }
    } else {
      const pos = (p: BookRange["start"]) =>
        `${p.book} ${p.chapter}${p.verse !== null ? `:${p.verse}` : ""}`;
      units.push({
        kind: "range",
        key: `range-${r.id}`,
        label: `${pos(r.start)} \u2013 ${pos(r.end)}`,
        range: r,
      });
    }
  }

  return units;
}

export function MemoryPage() {
  const { data: books = [] } = useBooks();

  const [ranges, setRanges] = useState<BookRange[]>([]);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [results, setResults] = useState<Record<string, ChapterResult>>({});

  const updateRange = (id: number, patch: Partial<BookRange>) =>
    setRanges((prev) => prev.map((r) => (r.id === id ? { ...r, ...patch } : r)));

  const removeRange = (id: number) =>
    setRanges((prev) => prev.filter((r) => r.id !== id));

  const addRange = (book: BookInfo) => {
    const nextId = ranges.length ? Math.max(...ranges.map((r) => r.id)) + 1 : 1;
    const pos = { book: book.book, bookId: book.id, chapter: 1, verse: null };
    setRanges((prev) => [
      ...prev,
      { id: nextId, start: { ...pos }, end: { ...pos, chapter: book.chapters } },
    ]);
  };

  const memoryUnits = buildMemoryUnits(ranges);

  async function handleCheckAll() {
    const scriptureRanges = memoryUnits.map((u) =>
      u.kind === "chapter"
        ? {
            start: { book: u.bookId, chapter: u.chapter, verse: null },
            end: { book: u.bookId, chapter: u.chapter, verse: null },
          }
        : {
            start: {
              book: u.range.start.bookId,
              chapter: u.range.start.chapter,
              verse: u.range.start.verse,
            },
            end: {
              book: u.range.end.bookId,
              chapter: u.range.end.chapter,
              verse: u.range.end.verse,
            },
          }
    );

    const answersList = memoryUnits.map((u) => answers[u.key] ?? "");

    const res = await fetch(`${API_URL}/check`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        scripture: { translation: "KJV", ranges: scriptureRanges },
        answers: answersList,
      }),
    });

    if (!res.ok) {
      console.error("Check failed:", res.status);
      return;
    }

    const resultsList: ChapterResult[] = await res.json();

    const resultsByUnit: Record<string, ChapterResult> = {};
    memoryUnits.forEach((u, i) => {
      resultsByUnit[u.key] = resultsList[i];
    });

    setResults(resultsByUnit);
  }

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div className="flex flex-row flex-wrap gap-3 items-start">
          {ranges.map((r) => (
            <RangeCard
              key={r.id}
              range={r}
              books={books}
              onUpdate={updateRange}
              onRemove={removeRange}
            />
          ))}
          <select
            value=""
            onChange={(e) => {
              const selected = books.find((b) => b.id === Number(e.target.value));
              if (selected) addRange(selected);
            }}
            disabled={books.length === 0}
            className="appearance-none h-10 rounded-lg border border-border px-4 text-sm text-muted-foreground bg-transparent focus:outline-none focus:ring-2 focus:ring-ring cursor-pointer"
            >
            <option value="" disabled>+ Add</option>
            {books.map((b) => (
              <option key={b.id} value={b.id}>{b.book}</option>
            ))}
          </select>
        </div>

        <div className="flex flex-col gap-4">
          {memoryUnits.map((u) => (
            <MemoryCard
              key={u.key}
              sectionTitle={u.label}
              value={answers[u.key] ?? ""}
              onChange={(value) => setAnswers((prev) => ({ ...prev, [u.key]: value }))}
              result={results[u.key]}
            />
          ))}
        </div>
      </div>

      <div className="border-t bg-background px-6 py-3 flex justify-end shrink-0">
        <Button onClick={handleCheckAll}>Check</Button>
      </div>
    </div>
  );
}