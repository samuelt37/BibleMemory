import { API_URL } from "../constants/config";
import type { BookInfo } from "../models/BookInfo";

export async function getBooks(): Promise<BookInfo[]> {
  const response = await fetch(`${API_URL}/books`);

  if (!response.ok) {
    throw new Error("Failed to fetch books");
  }

  return response.json();
}