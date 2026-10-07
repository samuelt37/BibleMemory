import {
  createContext,
  useContext,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type { BookRange } from "../types";

type ReviewSessionContextType = {
  ranges: BookRange[];
  setRanges: React.Dispatch<React.SetStateAction<BookRange[]>>;
  addRangeDirectly: (range: BookRange) => void;
  loadRanges: (ranges: BookRange[]) => void;
  loadVersion: number; // bumps only when loadRanges is called
  newRangeId: () => number;
};

const ReviewSessionContext = createContext<ReviewSessionContextType | null>(
  null,
);

export function ReviewSessionProvider({ children }: { children: ReactNode }) {
  const [ranges, setRanges] = useState<BookRange[]>([]);
  const [loadVersion, setLoadVersion] = useState(0);
  const nextIdRef = useRef(1);
  const newRangeId = () => nextIdRef.current++;

  const addRangeDirectly = (range: BookRange) => {
    const id = nextIdRef.current++;
    setRanges((prev) => [...prev, { ...range, id }]);
  };

  const loadRanges = (newRanges: BookRange[]) => {
    setRanges(newRanges.map((r, i) => ({ ...r, id: i + 1 })));
    nextIdRef.current = newRanges.length + 1;
    setLoadVersion((v) => v + 1);
  };

  return (
    <ReviewSessionContext.Provider
      value={{
        ranges,
        setRanges,
        addRangeDirectly,
        loadRanges,
        loadVersion,
        newRangeId,
      }}
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
