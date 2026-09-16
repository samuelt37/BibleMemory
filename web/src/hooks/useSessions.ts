// hooks/useSessions.ts
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@clerk/clerk-react";
import { API_URL } from "@/constants/config";
import type { ScriptureRangeDTO } from "@/models/BookRange";
import type { MemorySession } from "@/models/Session";


export function useSessionHistory() {
  const { getToken, isSignedIn } = useAuth();

  return useQuery({
    queryKey: ["sessions", "history"],
    queryFn: async () => {
      const token = await getToken();
      const res = await fetch(`${API_URL}/sessions/history`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Failed to fetch history");
      return res.json() as Promise<MemorySession[]>;
    },
    enabled: !!isSignedIn,
  });
}

export function useSessionBookmarks() {
  const { getToken, isSignedIn } = useAuth();

  return useQuery({
    queryKey: ["sessions", "bookmarks"],
    queryFn: async () => {
      const token = await getToken();
      const res = await fetch(`${API_URL}/sessions/bookmarks`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Failed to fetch bookmarks");
      return res.json() as Promise<MemorySession[]>;
    },
    enabled: !!isSignedIn,
  });
}

export function useSaveSession() {
  const { getToken } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (ranges: ScriptureRangeDTO[]) => {
      const token = await getToken();
      const res = await fetch(`${API_URL}/sessions`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ ranges }),
      });
      if (!res.ok) throw new Error("Failed to save session");
      return res.json() as Promise<MemorySession>;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions", "history"] });
    },
  });
}

export function useDeleteSession() {
  const { getToken } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (sessionId: number) => {
      const token = await getToken();
      const res = await fetch(`${API_URL}/sessions/${sessionId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Failed to delete session");
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions", "history"] });
      queryClient.invalidateQueries({ queryKey: ["sessions", "bookmarks"] });
    },
  });
}