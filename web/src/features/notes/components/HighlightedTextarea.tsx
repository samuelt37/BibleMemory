import { useRef } from "react";

export function HighlightedTextarea({
  value,
  onChange,
  highlight,
  markClass,
}: {
  value: string;
  onChange: (v: string) => void;
  highlight?: string;
  markClass?: string;
}) {
  const backdropRef = useRef<HTMLDivElement>(null);
  const idx = highlight ? value.indexOf(highlight) : -1;

  return (
    <div className="relative mt-2">
      <div
        ref={backdropRef}
        aria-hidden
        className="pointer-events-none absolute inset-0 overflow-hidden whitespace-pre-wrap break-words rounded-md border border-transparent p-2 text-sm text-transparent"
      >
        {idx >= 0 ? (
          <>
            {value.slice(0, idx)}
            <mark className={`rounded text-transparent ${markClass ?? ""}`}>
              {highlight}
            </mark>
            {value.slice(idx + highlight!.length)}
          </>
        ) : (
          value
        )}
        {"\n"}
      </div>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onScroll={(e) => {
          if (backdropRef.current)
            backdropRef.current.scrollTop = e.currentTarget.scrollTop;
        }}
        placeholder="Summarize the section in your own words..."
        rows={4}
        className="relative w-full resize-none rounded-md border bg-transparent p-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
      />
    </div>
  );
}
