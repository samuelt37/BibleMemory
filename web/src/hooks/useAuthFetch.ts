// hooks/useAuthFetch.ts
import { useAuth } from "@clerk/clerk-react";
import { API_URL } from "@/constants/config";

export function useAuthFetch() {
  const { getToken } = useAuth();

  return async (path: string, options: RequestInit = {}) => {
    const token = await getToken();
    return fetch(`${API_URL}${path}`, {
      ...options,
      headers: {
        ...options.headers,
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
    });
  };
}