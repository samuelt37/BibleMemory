// pages/MemoryPage.tsx
import { useState } from "react";
import { MemoryCard } from "../components/MemoryCard.tsx";
import { RangeCard } from "../components/RangeCard.tsx";
import { useBooks } from "@/hooks/Books.ts";
import { Button } from "@/components/ui/button";
import { API_URL } from "@/constants/config";
import type { BookRange } from "@/models/BookRange";
import type { BookInfo } from "@/models/BookInfo.ts";

type ChapterEntry = { book: string; chapter: number };
type ChapterResult = { accuracy: number; feedback: string };

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
  setRanges((prev) => [
    ...prev,
    { id: nextId, book: book.book, bookId: book.id, from: 1, to: book.chapters },
  ]);
};

  const key = (e: ChapterEntry) => `${e.book}-${e.chapter}`;

  // Flatten all ranges into individual chapter entries, deduping overlaps
  const chapterEntries: ChapterEntry[] = Array.from(
    new Map(
      ranges.flatMap((r) =>
        Array.from({ length: Math.max(r.to - r.from + 1, 0) }, (_, i) => {
          const entry = { book: r.book, chapter: r.from + i };
          return [key(entry), entry] as const;
        })
      )
    ).values()
  );

  async function handleCheckAll() {
    const scriptureRanges = chapterEntries.map((e) => {
      const bookIndex = books.findIndex((b) => b.book === e.book);
      return {
        start: { book: bookIndex, chapter: e.chapter, verse: null },
        end: { book: bookIndex, chapter: e.chapter, verse: null },
      };
    });

    const answersList = chapterEntries.map((e) => answers[key(e)] ?? "");

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

    const resultsByChapter: Record<string, ChapterResult> = {};
    chapterEntries.forEach((e, i) => {
      resultsByChapter[key(e)] = resultsList[i];
    });

    setResults(resultsByChapter);
  }

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div className="flex flex-row gap-3 items-start">
          {ranges.map((r) => (
            <RangeCard
              range={r}
              books={books}
              onUpdate={updateRange}
              onRemove={removeRange}
            />
          ))}
          <select
            value=""
            onChange={(e) => {
              if (e.target.value) {
                const book = books.find((b) => b.book === e.target.value);
                if (book) {
                  addRange(book);
                }
              }
            }}
            disabled={books.length === 0}
            className="appearance-none [&::-ms-expand]:hidden h-10 rounded-lg border border-border px-4 text-sm text-muted-foreground bg-transparent focus:outline-none focus:ring-2 focus:ring-ring cursor-pointer">
            <option value="" disabled>
              + Add range
            </option>
            {books.map((b) => (
              <option key={b.book} value={b.book}>
                {b.book}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-col gap-4">
          {chapterEntries.map((e) => (
            <MemoryCard
              key={key(e)}
              sectionTitle={`${e.book} — Chapter ${e.chapter}`}
              value={answers[key(e)] ?? ""}
              onChange={(value) =>
                setAnswers((prev) => ({ ...prev, [key(e)]: value }))
              }
              result={results[key(e)]}
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