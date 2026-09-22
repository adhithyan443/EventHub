import { Outlet, useLocation } from "react-router-dom";
import OrganizerSidebar from "./OrganizerSidebar";
import OrganizerTopBar from "./OrganizerTopBar";
import useOrganizerStore from "../../store/organizerStore";
import { useEffect, useState } from "react";

export default function OrganizerLayout() {
  const location = useLocation();
  const isCreateFlow = location.pathname.startsWith("/organizer/events/create");
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const fetchProfile = useOrganizerStore((state) => state.fetchProfile);

  useEffect(() => {
    fetchProfile();
  }, [fetchProfile]);

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#f9f9ff] text-[#141b2b] relative">
      {/* Sidebar (supports static on lg+ and drawer overlay on mobile) */}
      <OrganizerSidebar isOpen={sidebarOpen} onClose={() => setSidebarOpen(false)} />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* TopBar (Hide "+ Create Event" button when already inside the event creation flow) */}
        <OrganizerTopBar
          showCreateButton={!isCreateFlow}
          onMenuToggle={() => setSidebarOpen((prev) => !prev)}
        />

        {/* Page Content Outlet */}
        <main className="flex-1 overflow-y-auto bg-[#f9f9ff] relative">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
