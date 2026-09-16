import { useQuery } from "@tanstack/react-query";
import { API_URL } from "@/constants/config";
import type { BookInfo } from "@/models/BookInfo";

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
    queryFn: async () => {
      const res = await fetch(`${API_URL}/books/${bookName}/chapters`);
      if (!res.ok) throw new Error(`Request failed: ${res.status}`);
      return res.json();
    },
    enabled: !!bookName,
    staleTime: Infinity,
  });
}