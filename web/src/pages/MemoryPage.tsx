// pages/MemoryPage.tsx
import { useState } from "react";
import { MemoryCard } from "../components/MemoryCard.tsx";
import { RangeCard } from "../components/RangeCard.tsx";
import { useBooks } from "@/hooks/Books.ts";
import { Button } from "@/components/ui/button";
import { API_URL } from "@/constants/config";
import type { BookRange } from "@/models/BookRange";

type ChapterEntry = { book: string; chapter: number };
type ChapterResult = { accuracy: number; feedback: string };

export function MemoryPage() {
  const { data: books = [] } = useBooks();
  // const { data: chapterCounts = {} } = useBookChapterCounts();
  const chapterCounts: Record<string, number> = Object.fromEntries(
    books.map((b) => [b, 10])
  );

  const [ranges, setRanges] = useState<BookRange[]>(() =>
    books.length
      ? [{ id: 1, book: books[0], from: 1, to: chapterCounts[books[0]] ?? 1 }]
      : []
  );
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [results, setResults] = useState<Record<string, ChapterResult>>({});

  const updateRange = (id: number, patch: Partial<BookRange>) =>
    setRanges((prev) => prev.map((r) => (r.id === id ? { ...r, ...patch } : r)));

  const removeRange = (id: number) =>
    setRanges((prev) => prev.filter((r) => r.id !== id));

  const addRange = () => {
    const nextId = ranges.length ? Math.max(...ranges.map((r) => r.id)) + 1 : 1;
    const defaultBook = ranges[ranges.length - 1]?.book ?? books[0];
    setRanges((prev) => [
      ...prev,
      { id: nextId, book: defaultBook, from: 1, to: 1 },
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
      const bookIndex = books.indexOf(e.book);
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
        <div className="flex flex-col gap-3">
          {ranges.map((r) => (
            <RangeCard
              key={r.id}
              range={r}
              books={books}
              chapterCounts={chapterCounts}
              onUpdate={updateRange}
              onRemove={removeRange}
            />
          ))}
          <Button variant="outline" onClick={addRange}>
            + Add range
          </Button>
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