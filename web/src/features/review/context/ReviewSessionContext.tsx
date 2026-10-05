import { createContext, useContext, useState, type ReactNode } from "react";
import type { BookRange } from "../types";

type ReviewSessionContextType = {
  ranges: BookRange[];
  setRanges: React.Dispatch<React.SetStateAction<BookRange[]>>;
  addRangeDirectly: (range: BookRange) => void;
  loadRanges: (ranges: BookRange[]) => void;
  loadVersion: number; // bumps only when loadRanges is called
};

const ReviewSessionContext = createContext<ReviewSessionContextType | null>(
  null,
);

export function ReviewSessionProvider({ children }: { children: ReactNode }) {
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
    <ReviewSessionContext.Provider
      value={{ ranges, setRanges, addRangeDirectly, loadRanges, loadVersion }}
    >
      {children}
    </ReviewSessionContext.Provider>
  );
}

export function useReviewSession() {
  const ctx = useContext(ReviewSessionContext);
  if (!ctx)
    throw new Error(
      "useReviewSession must be used within ReviewSessionProvider",
    );
  return ctx;
}
