import { useState } from "react";
import { Outlet } from "react-router-dom";
import AdminSidebar from "./AdminSidebar";
import AdminTopBar from "./AdminTopBar";

export default function AdminLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#f1f3ff] text-[#141b2b]">
      {/* Sidebar */}
      <AdminSidebar
        isOpen={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* TopBar */}
        <AdminTopBar
          onMenuToggle={() => setSidebarOpen((prev) => !prev)}
        />

        {/* Page Content Outlet */}
        <main className="flex-1 flex overflow-hidden bg-white">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
