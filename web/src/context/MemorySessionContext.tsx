import { createContext, useContext, useState, type ReactNode } from "react";
import type { BookRange } from "@/models/BookRange";

type MemorySessionContextType = {
  ranges: BookRange[];
  setRanges: React.Dispatch<React.SetStateAction<BookRange[]>>;
  addRangeDirectly: (range: BookRange) => void;
  loadRanges: (ranges: BookRange[]) => void;
  loadVersion: number; // bumps only when loadRanges is called
};

const MemorySessionContext = createContext<MemorySessionContextType | null>(null);

export function MemorySessionProvider({ children }: { children: ReactNode }) {
  const [ranges, setRanges] = useState<BookRange[]>([]);
  const [loadVersion, setLoadVersion] = useState(0);

  const addRangeDirectly = (range: BookRange) => {
    setRanges((prev) => {
      const nextId = prev.length ? Math.max(...prev.map((r) => r.id)) + 1 : 1;
      return [...prev, { ...range, id: nextId }];
    });
  };

  const loadRanges = (newRanges: BookRange[]) => {
    setRanges(newRanges.map((r, i) => ({ ...r, id: i + 1 })));
    setLoadVersion((v) => v + 1);
  };

  return (
    <MemorySessionContext.Provider
      value={{ ranges, setRanges, addRangeDirectly, loadRanges, loadVersion }}
    >
      {children}
    </MemorySessionContext.Provider>
  );
}

export function useMemorySession() {
  const ctx = useContext(MemorySessionContext);
  if (!ctx) throw new Error("useMemorySession must be used within MemorySessionProvider");
  return ctx;
}