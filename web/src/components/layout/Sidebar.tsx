import { useState } from "react";
import { LogIn } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { SignedIn, SignedOut, SignInButton, UserButton } from "@clerk/clerk-react";



export function Sidebar() {
  const [open, setOpen] = useState(true);

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