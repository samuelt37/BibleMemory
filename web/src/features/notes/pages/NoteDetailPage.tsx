import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ChevronLeft } from "lucide-react";
import { useAuth } from "@clerk/clerk-react";

import { NoteView } from "../components/NoteView";
import type { Chunk } from "@/features/references/types";
import { RefsBox } from "@/features/references/components/RefBox";
import { RetagButton } from "@/features/references/components/RetagButton";
import { API_URL } from "@/constants/config";

export function NoteDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { getToken } = useAuth();

  const [refreshKey, setRefreshKey] = useState(0);

  const [chunks, setChunks] = useState<Chunk[]>([]);
  const [chunksLoading, setChunksLoading] = useState(true);

  const loadChunks = useCallback(async () => {
    if (!id) return;
    setChunksLoading(true);
    try {
      const token = await getToken();
      const res = await fetch(`${API_URL}/notes/${id}/chunks`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) setChunks(await res.json());
    } finally {
      setChunksLoading(false);
    }
  }, [id, getToken]);

  useEffect(() => {
    loadChunks();
  }, [loadChunks]);

  async function handleRetagged() {
    await loadChunks();
    setRefreshKey((k) => k + 1);
  }

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div className="flex items-start justify-between">
          <button
            type="button"
            onClick={() => navigate(`/notes${location.search}`)}
            className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground w-fit"
          >
            <ChevronLeft size={16} />
            Back to notes
          </button>

          {id && <RetagButton noteId={Number(id)} onDone={handleRetagged} />}
        </div>

        {id ? (
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start">
            <div className="min-w-0 flex-1">
              <NoteView key={refreshKey} noteId={Number(id)} />
            </div>
            <RefsBox chunks={chunks} loading={chunksLoading} />
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">Note not found.</p>
        )}
      </div>
    </div>
  );
}
