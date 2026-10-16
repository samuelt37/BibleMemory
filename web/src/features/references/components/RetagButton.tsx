import { useState } from "react";
import { RefreshCw } from "lucide-react";
import { useAuth } from "@clerk/clerk-react";
import { rereferenceNote } from "../api";

type Props = {
  noteId: number;
  onDone: () => void | Promise<void>;
};

export function RetagButton({ noteId, onDone }: Props) {
  const { getToken } = useAuth();
  const [retagging, setRetagging] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    if (retagging) return;
    setRetagging(true);
    setError(null);
    try {
      await rereferenceNote(noteId, await getToken());
      await onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Retag failed");
    } finally {
      setRetagging(false);
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <button
        type="button"
        onClick={handleClick}
        disabled={retagging}
        className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground disabled:opacity-50"
      >
        <RefreshCw size={14} className={retagging ? "animate-spin" : ""} />
        {retagging ? "Retagging..." : "Retag verses"}
      </button>
      {error && <p className="text-xs text-red-600">{error}</p>}
    </div>
  );
}
