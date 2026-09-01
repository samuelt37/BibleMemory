import { useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  TooltipProvider,
} from "@/components/ui/tooltip";
import {
  Collapsible,
  CollapsibleTrigger,
  CollapsibleContent,
} from "@/components/ui/collapsible";
import { ChevronLeft, ChevronRight, ChevronDown, BookOpen } from "lucide-react";
import { useBooks } from "@/hooks/Books";

const NEW_TESTAMENT_START = "Matthew";
function groupBooks(books: string[]) {
  const splitIndex = books.indexOf(NEW_TESTAMENT_START);
  if (splitIndex === -1) return { oldTestament: books, newTestament: [] };

  return {
    oldTestament: books.slice(0, splitIndex),
    newTestament: books.slice(splitIndex),
  };
}

export function Sidebar() {
  const [open, setOpen] = useState(true);
  const [otOpen, setOtOpen] = useState(true);
  const [ntOpen, setNtOpen] = useState(true);

  // const { data: books = [], isLoading, error } = useBooks();
  // const { oldTestament, newTestament } = groupBooks(books);

  function renderBookLink(book: string) {
    const href = `/book/${encodeURIComponent(book)}`;

    return open ? (
      <Link
        key={book}
        to={href}
        className="flex items-center gap-3 rounded-md px-3 py-2 text-sm hover:bg-muted"
      >
        <BookOpen size={16} className="shrink-0" />
        <span className="truncate">{book}</span>
      </Link>
    ) : (
      <Tooltip key={book}>
        <TooltipTrigger
          render={
            <Link
              to={href}
              className="flex items-center justify-center rounded-md p-2 hover:bg-muted"
            />
          }
        >
          <BookOpen size={16} />
        </TooltipTrigger>
        <TooltipContent side="right">{book}</TooltipContent>
      </Tooltip>
    );
  }

  function renderSection(
    title: string,
    isOpen: boolean,
    setIsOpen: (v: boolean) => void,
    sectionBooks: string[]
  ) {
    // when the whole sidebar is collapsed (icon-only), skip the group header/toggle entirely
    if (!open) return sectionBooks.map(renderBookLink);

    return (
      <Collapsible open={isOpen} onOpenChange={setIsOpen} key={title}>
        <CollapsibleTrigger className="flex w-full items-center justify-between px-3 pt-3 pb-1 text-xs font-semibold uppercase text-muted-foreground hover:text-foreground">
          {title}
          <ChevronDown
            size={14}
            className={`transition-transform ${isOpen ? "rotate-0" : "-rotate-90"}`}
          />
        </CollapsibleTrigger>
        <CollapsibleContent className="flex flex-col gap-1">
          {sectionBooks.map(renderBookLink)}
        </CollapsibleContent>
      </Collapsible>
    );
  }

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

        {/* <nav className="flex flex-col gap-1 overflow-y-auto p-2">
          {isLoading && open && (
            <span className="px-3 py-2 text-sm text-muted-foreground">Loading…</span>
          )}
          {error && open && (
            <span className="px-3 py-2 text-sm text-destructive">Couldn't load books</span>
          )}

          {renderSection("Old Testament", otOpen, setOtOpen, oldTestament)}
          {renderSection("New Testament", ntOpen, setNtOpen, newTestament)} */}
        {/* </nav> */}
      </div>
    </TooltipProvider>
  );
}