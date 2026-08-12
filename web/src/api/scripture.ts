import { API_URL } from "../constants/config";
import type { Book } from "../models/Book";

export async function getBooks(): Promise<Book[]> {
  const response = await fetch(`${API_URL}/books`);

  if (!response.ok) {
    throw new Error("Failed to fetch books");
  }

  return response.json();
}