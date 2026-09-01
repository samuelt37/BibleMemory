import { useQuery } from "@tanstack/react-query";
import { API_URL } from "@/constants/config";
import type { BookInfo } from "@/models/Book";

async function fetchBooks(): Promise<BookInfo[]> {
  const res = await fetch(`${API_URL}/books`);
  if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  return res.json();
}

async function fetchChapters(bookName: string): Promise<number> {
  const res = await fetch(`${API_URL}/books/${bookName}/chapters`);
  if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  return res.json();
}

export function useBooks() {
  return useQuery<BookInfo[]>({
    queryKey: ["books"],
    queryFn: async () => {
      const res = await fetch(`${API_URL}/books`);
      if (!res.ok) throw new Error("Failed to fetch books");
      return res.json();
    },
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