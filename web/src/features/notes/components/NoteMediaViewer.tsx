import { forwardRef, useState } from "react";
import type { NoteMedia } from "../types";

export const NoteMediaViewer = forwardRef<
  HTMLVideoElement,
  { media: NoteMedia }
>(({ media }, ref) => {
  const [failed, setFailed] = useState(false);

  if (failed) {
    return (
      <p className="rounded-lg border p-3 text-sm text-muted-foreground">
        This {media.type} can't be previewed in your browser.
      </p>
    );
  }

  if (media.type === "video") {
    return (
      <video
        ref={ref}
        src={media.url}
        controls
        preload="metadata"
        playsInline
        onError={() => setFailed(true)}
        className="w-full max-h-[50vh] rounded-lg bg-black"
      />
    );
  }

  return (
    <img
      src={media.url}
      alt=""
      onError={() => setFailed(true)}
      className="w-full max-h-[60vh] rounded-lg object-contain"
    />
  );
});
NoteMediaViewer.displayName = "NoteMediaViewer";
