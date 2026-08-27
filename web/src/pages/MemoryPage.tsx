// pages/MemoryPage.tsx
import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { MemoryCard } from "../components/MemoryCard.tsx";
import { useChapters } from "@/hooks/Books.ts";
import { Button } from "@/components/ui/button";
import { useBooks } from "@/hooks/Books.ts";

export function MemoryPage() {
  const { bookName } = useParams();
  const { data: books = [] } = useBooks(); 
  const { data: totalChapters = 0, isLoading, error } = useChapters(bookName);

  const [startChapter, setStartChapter] = useState(1);
  const [endChapter, setEndChapter] = useState(1);
  const [answers, setAnswers] = useState<Record<number, string>>({});

  const [results, setResults] = useState<Record<number, { accuracy: number; feedback: string }>>({});

  useEffect(() => {
    setStartChapter(1);
    setEndChapter(totalChapters);
    setAnswers({});
  }, [bookName, totalChapters]);

  const effectiveEnd = Math.min(endChapter || totalChapters, totalChapters);

  if (isLoading) return <p className="p-6">Loading…</p>;
  if (error) return <p className="p-6 text-destructive">Couldn't load chapters</p>;

  const chapterNumbers = Array.from(
    { length: Math.max(effectiveEnd - startChapter + 1, 0) },
    (_, i) => startChapter + i
  );

  async function handleCheckAll() {
  const bookIndex = books.indexOf(bookName!);
  const ranges = chapterNumbers.map((chapterNum) => ({
    start: { book: bookIndex, chapter: chapterNum, verse: null },
    end: { book: bookIndex, chapter: chapterNum, verse: null },
  }));

  const answersList = chapterNumbers.map((chapterNum) => answers[chapterNum] ?? "");

  const res = await fetch("/check", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      scripture: { translation: "KJV", ranges },
      answers: answersList,
    }),
  });

  if (!res.ok) {
    console.error("Check failed:", res.status);
    return;
  }

  const results: { accuracy: number; feedback: string }[] = await res.json();

  // results[i] corresponds to chapterNumbers[i] — zip them back together
  const resultsByChapter: Record<number, { accuracy: number; feedback: string }> = {};
  chapterNumbers.forEach((chapterNum, i) => {
    resultsByChapter[chapterNum] = results[i];
  });

  setResults(resultsByChapter);
}

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div className="flex items-center gap-2 text-sm">
          <label className="flex items-center gap-1">
            From
            <input
              type="number"
              min={1}
              max={totalChapters}
              value={startChapter}
              onChange={(e) => setStartChapter(Number(e.target.value))}
              className="w-16 rounded border px-2 py-1"
            />
          </label>
          <label className="flex items-center gap-1">
            To
            <input
              type="number"
              min={1}
              max={totalChapters}
              value={endChapter || totalChapters}
              onChange={(e) => setEndChapter(Number(e.target.value))}
              className="w-16 rounded border px-2 py-1"
            />
          </label>
          <span className="text-muted-foreground">of {totalChapters} chapters</span>
        </div>

        <div className="flex flex-col gap-4">
          {chapterNumbers.map((chapterNum) => (
            <MemoryCard
              key={chapterNum}
              sectionTitle={`Chapter ${chapterNum}`}
              value={answers[chapterNum] ?? ""}
              onChange={(value) => setAnswers((prev) => ({ ...prev, [chapterNum]: value }))}
              result={results[chapterNum]}
            />
          ))}
        </div>
      </div>

      <div className="border-t bg-background px-6 py-3 flex justify-end shrink-0">
        <Button onClick={handleCheckAll}>Check All</Button>
      </div>
    </div>
  );
}