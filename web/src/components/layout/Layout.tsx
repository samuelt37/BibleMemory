import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { MemorySessionProvider } from "@/context/MemorySessionContext";

export function Layout() {
  return (
    <MemorySessionProvider>
      <div className="flex h-screen overflow-hidden">
        <Sidebar />
        <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
          <Outlet />
        </main>
      </div>
    </MemorySessionProvider>
  );
}