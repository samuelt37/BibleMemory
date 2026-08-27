import { useQuery } from "@tanstack/react-query";

async function fetchBooks(): Promise<string[]> {
  const res = await fetch("/books");
  if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  return res.json();
}

async function fetchChapters(bookName: string): Promise<number> {
  const res = await fetch(`/books/${bookName}/chapters`);
  if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  return res.json();
}

export function useBooks() {
  return useQuery({
    queryKey: ["books"],
    queryFn: fetchBooks,
    staleTime: Infinity, // book list never changes — never auto-refetch
  });
}

export function useChapters(bookName: string | undefined) {
  return useQuery({
    queryKey: ["chapters", bookName],
    queryFn: () => fetchChapters(bookName!),
    enabled: !!bookName,
    staleTime: Infinity, // chapter list never changes — never auto-refetch
  });
}