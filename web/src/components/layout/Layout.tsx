import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { ReviewSessionProvider } from "@/features/review";

export function Layout() {
  return (
    <ReviewSessionProvider>
      <div className="flex h-screen overflow-hidden">
        <Sidebar />
        <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
          <Outlet />
        </main>
      </div>
    </ReviewSessionProvider>
  );
}
