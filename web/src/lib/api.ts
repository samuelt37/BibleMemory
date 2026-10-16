// lib/api.ts  (shared)
import { API_URL } from "@/constants/config";

export async function apiFetch(path: string, token: string | null, init: RequestInit = {}) {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      ...init.headers,
      Authorization: `Bearer ${token}`,
      ...(init.body ? { "Content-Type": "application/json" } : {}),
    },
  });
  if (!res.ok) throw new Error(await res.text());
  return res;
}