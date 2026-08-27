// components/MemorySection.tsx
type MemoryCardProps = {
  sectionTitle: string;
  value: string;
  onChange: (value: string) => void;
  result?: { accuracy: number; feedback: string };
};

export function MemoryCard({ sectionTitle, value, onChange, result }: MemoryCardProps) {
  return (
    <div className="p-4 border rounded-lg bg-card text-card-foreground shadow-xs">
      <p className="italic">{sectionTitle}</p>

      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="Type the verse from memory…"
        rows={4}
        className="mt-2 w-full resize-none rounded-md border p-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
      />

      {result && (
        <p className={`mt-2 text-sm ${result.accuracy >= 7 ? "text-green-600" : result.accuracy >= 4 ? "text-yellow-600" : "text-red-600"}`}>
          {result.accuracy}/10 — {result.feedback}
        </p>
      )}
    </div>
  );
}