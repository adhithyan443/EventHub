import { Outlet, useLocation } from "react-router-dom";
import OrganizerSidebar from "./OrganizerSidebar";
import OrganizerTopBar from "./OrganizerTopBar";
import useOrganizerStore from "../../store/organizerStore";
import { useEffect } from "react";

export default function OrganizerLayout() {
  const location = useLocation();
  const isCreateFlow = location.pathname.startsWith("/organizer/events/create");

  const fetchProfile = useOrganizerStore((state) => state.fetchProfile);

  useEffect(() => {
     fetchProfile();
    // console.log(profile);
  }, [fetchProfile]);


  // useEffect(() => {
  //   let isMounted = true;

  //   const loadData = async () => {
  //     const data = await fetchProfile();
  //     if (isMounted && data) {
  //       console.log('Resolved profile:', data);
  //     }
  //   };

  //   loadData();

  //   return () => {
  //     isMounted = false;
  //   };
  // }, [fetchProfile])


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
