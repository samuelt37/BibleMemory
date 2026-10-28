// features/scripture/components/BookSelect.tsx
import type { BookInfo } from "../types"; // adjust path

type Props = {
  books: BookInfo[];
  value: number | null;
  onChange: (book: BookInfo | null) => void;
  placeholder?: string; // when set, adds an empty option that returns null
  className?: string;
  noteCounts?: Record<number, number>; // bookId -> number of notes
};

export function BookSelect({
  books,
  value,
  onChange,
  placeholder,
  className,
  noteCounts,
}: Props) {
  return (
    <select
      value={value ?? ""}
      onChange={(e) => {
        const id = e.target.value;
        onChange(
          id === "" ? null : (books.find((b) => b.id === Number(id)) ?? null),
        );
      }}
      className={className ?? "rounded-md border bg-background p-1 text-sm"}
    >
      {placeholder !== undefined && <option value="">{placeholder}</option>}
      {books.map((b) => {
        const count = noteCounts?.[b.id];
        return (
          <option key={b.id} value={b.id}>
            {count !== undefined ? `${b.book} (${count})` : b.book}
          </option>
        );
      })}
    </select>
  );
}
