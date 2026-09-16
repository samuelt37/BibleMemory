import { useState } from "react";
import { ChevronLeft, ChevronRight, ChevronDown, LogIn, History, Bookmark, X } from "lucide-react";
import { SignedIn, SignedOut, SignInButton, UserButton } from "@clerk/clerk-react";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useMemorySession } from "@/context/MemorySessionContext";
import { useBooks } from "@/hooks/Books";
import type { BookRange, ScriptureRangeDTO } from "@/models/BookRange";
import type { BookInfo } from "@/models/BookInfo";
import type { MemorySession } from "@/models/Session";
import { useDeleteSession, useSessionBookmarks, useSessionHistory } from "@/hooks/useSessions";

function sessionLabel(session: MemorySession, books: BookInfo[]): string {
  return session.ranges
    .map((dto) => {
      const book = books.find((b) => b.id === dto.startBookId)?.book ?? "";
      return dto.startChapter === dto.endChapter
        ? `${book} ${dto.startChapter}`
        : `${book} ${dto.startChapter}\u2013${dto.endChapter}`;
    })
    .join(", ");
}

function DropdownSection({
  label,
  icon,
  items,
  books,
  open,
  sidebarOpen,
  onToggle,
  onSelect,
  onDelete,
}: {
  label: string;
  icon: React.ReactNode;
  items: MemorySession[];
  books: BookInfo[];
  open: boolean;
  sidebarOpen: boolean;
  onToggle: () => void;
  onSelect: (item: MemorySession) => void;
  onDelete: (item: MemorySession) => void;
}) {
  const safeItems = items ?? [];
  if (!sidebarOpen) {
    // collapsed sidebar — just show the icon, no dropdown
    return (
      <button
        type="button"
        className="flex items-center justify-center rounded-md p-2 hover:bg-muted"
        aria-label={label}
      >
        {icon}
      </button>
    );
  }

  return (
    <div className="flex flex-col">
      <button
        type="button"
        onClick={onToggle}
        className="flex items-center gap-2 rounded-md px-3 py-2 text-sm hover:bg-muted text-muted-foreground"
      >
        <ChevronDown
          size={14}
          className={`shrink-0 transition-transform ${open ? "" : "-rotate-90"}`}
        />
        <span className="font-medium">{label}</span>
      </button>

      {open && (
        <div className="ml-2 flex flex-col border-l pl-3">
          {safeItems.length === 0 ? (
            <p className="px-2 py-1.5 text-xs text-muted-foreground">Nothing yet</p>
          ) : (
            safeItems.map((item) => (
             <div key={item.id} className="group flex items-center gap-1 rounded-md hover:bg-muted">
              <button
                type="button"
                onClick={() => onSelect(item)}
                className="flex-1 truncate px-2 py-1.5 text-left text-sm"
              >
                {sessionLabel(item, books)}
              </button>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(item);
                }}
                className="mr-1 rounded p-1 text-muted-foreground opacity-0 group-hover:opacity-100 hover:bg-accent/20 hover:text-foreground"
                aria-label="Delete"
              >
                <X size={14} />
              </button>
             </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}

export function Sidebar() {
  const [open, setOpen] = useState(true);
  const [historyOpen, setHistoryOpen] = useState(true);
  const [bookmarksOpen, setBookmarksOpen] = useState(false);

  const { data: historyData } = useSessionHistory();
  const { data: bookmarksData } = useSessionBookmarks();
  const history = historyData ?? [];
  const bookmarks = bookmarksData ?? [];
  const { data: books = [] } = useBooks();
  const { loadRanges } = useMemorySession();
  const { mutate: deleteSession } = useDeleteSession();

  function dtoToBookRange(dto: ScriptureRangeDTO, id: number): BookRange {
    const startBook = books.find((b) => b.id === dto.startBookId);
    const endBook = books.find((b) => b.id === dto.endBookId);
    return {
      id,
      start: { book: startBook?.book ?? "", bookId: dto.startBookId, chapter: dto.startChapter, verse: dto.startVerse },
      end: { book: endBook?.book ?? "", bookId: dto.endBookId, chapter: dto.endChapter, verse: dto.endVerse },
    };
  }

  const handleSelect = (session: MemorySession) => {
    const bookRanges = session.ranges.map((dto, i) => dtoToBookRange(dto, i + 1));
    loadRanges(bookRanges);
  };

  const handleDelete = (session: MemorySession) => {
    deleteSession(session.id);
  };

  return (
    <TooltipProvider delay={0}>
      <div
        className={`flex h-screen flex-col border-r bg-background transition-all duration-300 ${
          open ? "w-56" : "w-16"
        }`}
      >
        <div className="flex items-center justify-between p-2">
          {open && <span className="px-2 font-semibold">BibleMemory</span>}
          <Button variant="ghost" size="icon" onClick={() => setOpen(!open)}>
            {open ? <ChevronLeft size={18} /> : <ChevronRight size={18} />}
          </Button>
        </div>

        <SignedIn>
          <div className="flex flex-col gap-1 p-2 overflow-y-auto">
            <DropdownSection
              label="Saved Sessions"
              icon={<Bookmark size={18} />}
              items={bookmarks}
              books={books}
              open={bookmarksOpen}
              sidebarOpen={open}
              onToggle={() => setBookmarksOpen((o) => !o)}
              onSelect={handleSelect}
              onDelete={handleDelete}
            />
            <DropdownSection
              label="Previous Sessions"
              icon={<History size={18} />}
              items={history}
              books={books}
              open={historyOpen}
              sidebarOpen={open}
              onToggle={() => setHistoryOpen((o) => !o)}
              onSelect={handleSelect}
              onDelete={handleDelete}
            />
          </div>
        </SignedIn>

        <div className="flex-1" />

        <div className="border-t p-2">
          <SignedOut>
            <SignInButton mode="modal">
              <Button variant="ghost" className="w-full justify-start gap-3 px-3">
                <LogIn size={18} className="shrink-0" />
                {open && <span>Log in</span>}
              </Button>
            </SignInButton>
          </SignedOut>
          <SignedIn>
            <UserButton />
          </SignedIn>
        </div>
      </div>
    </TooltipProvider>
  );
}