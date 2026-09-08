import { Outlet, useLocation } from "react-router-dom";
import OrganizerSidebar from "./OrganizerSidebar";
import OrganizerTopBar from "./OrganizerTopBar";

export default function OrganizerLayout() {
  const location = useLocation();
  const isCreateFlow = location.pathname.startsWith("/organizer/events/create");

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#f9f9ff] text-[#141b2b]">
      {/* Sidebar */}
      <OrganizerSidebar />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* TopBar (Hide "+ Create Event" button when already inside the event creation flow) */}
        <OrganizerTopBar showCreateButton={!isCreateFlow} />

        {/* Page Content Outlet */}
        <main className="flex-1 overflow-y-auto bg-[#f9f9ff] relative">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
