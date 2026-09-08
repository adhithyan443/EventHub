import { Link, useLocation } from "react-router-dom";
import { ORGANIZER_ROUTES } from "../../constants/eventConstants";
import imgOrganizerAvatar from "../../assets/organizer/1d07f03676805f8fc5a4e6b472edc608cc3dfc63.png";
import imgUserAvatar from "../../assets/organizer/77afb3678a691019904dbffccf7db02c242ed0f3.png";
import useAuthStore from "../../store/authStore";

export default function OrganizerSidebar({ className = "" }) {
  const location = useLocation();
  const user = useAuthStore((state) => state.user);

  const navItems = [
    {
      label: "Dashboard",
      path: ORGANIZER_ROUTES.DASHBOARD,
      exact: true,
      icon: (active) => (
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
          <path
            d="M2.25 2.25H7.5V7.5H2.25V2.25ZM10.5 2.25H15.75V7.5H10.5V2.25ZM2.25 10.5H7.5V15.75H2.25V10.5ZM10.5 10.5H15.75V15.75H10.5V10.5Z"
            fill={active ? "#00685F" : "#3D4947"}
          />
        </svg>
      ),
    },
    {
      label: "My Events",
      path: ORGANIZER_ROUTES.MY_EVENTS,
      icon: (active) => (
        <svg width="18" height="20" viewBox="0 0 18 20" fill="none">
          <path
            d="M15 3H14.25V1.5H12.75V3H5.25V1.5H3.75V3H3C2.17 3 1.5 3.67 1.5 4.5V16.5C1.5 17.33 2.17 18 3 18H15C15.83 18 16.5 17.33 16.5 16.5V4.5C16.5 3.67 15.83 3 15 3ZM15 16.5H3V6.75H15V16.5Z"
            fill={active ? "#00685F" : "#3D4947"}
          />
        </svg>
      ),
    },
    {
      label: "Create Event",
      path: ORGANIZER_ROUTES.CREATE_STEP_1,
      // Matches any step in the create-event flow
      isActive: (pathname) => pathname.startsWith("/organizer/events/create") || pathname === "/organizer/create-event",
      icon: (active) => (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M10 4.167V15.833M4.167 10H15.833"
            stroke={active ? "#00685F" : "#3D4947"}
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      ),
    },
    {
      label: "Bookings",
      path: ORGANIZER_ROUTES.BOOKINGS,
      icon: (active) => (
        <svg width="20" height="16" viewBox="0 0 20 16" fill="none">
          <path
            d="M18 0H2C0.9 0 0.01 0.9 0.01 2L0 14C0 15.1 0.89 16 1.99 16H18C19.1 16 20 15.1 20 14V2C20 0.9 19.1 0 18 0ZM18 14H2V8H18V14ZM18 4H2V2H18V4Z"
            fill={active ? "#00685F" : "#3D4947"}
          />
        </svg>
      ),
    },
    {
      label: "Attendees",
      path: ORGANIZER_ROUTES.ATTENDEES,
      icon: (active) => (
        <svg width="22" height="16" viewBox="0 0 22 16" fill="none">
          <path
            d="M15.5 8C17.43 8 18.99 6.43 18.99 4.5C18.99 2.57 17.43 1 15.5 1C13.57 1 12 2.57 12 4.5C12 6.43 13.57 8 15.5 8ZM6.5 8C8.43 8 9.99 6.43 9.99 4.5C9.99 2.57 8.43 1 6.5 1C4.57 1 3 2.57 3 4.5C3 6.43 4.57 8 6.5 8ZM6.5 10C3.83 10 0.5 11.34 0.5 14V16H12.5V14C12.5 11.34 9.17 10 6.5 10ZM15.5 10C15.14 10 14.73 10.03 14.3 10.09C15.59 11.02 16.5 12.33 16.5 14V16H21.5V14C21.5 11.34 18.17 10 15.5 10Z"
            fill={active ? "#00685F" : "#3D4947"}
          />
        </svg>
      ),
    },
    {
      label: "Reports",
      path: ORGANIZER_ROUTES.REPORTS,
      icon: (active) => (
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
          <path
            d="M15.75 14.25H2.25V2.25H0.75V15.75H15.75V14.25ZM4.5 12.75H6V6.75H4.5V12.75ZM8.25 12.75H9.75V3.75H8.25V12.75ZM12 12.75H13.5V9.75H12V12.75Z"
            fill={active ? "#00685F" : "#3D4947"}
          />
        </svg>
      ),
    },
    {
      label: "Settings",
      path: ORGANIZER_ROUTES.SETTINGS,
      icon: (active) => (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M16.2 11.8C16.3 11.2 16.4 10.6 16.4 10C16.4 9.4 16.3 8.8 16.2 8.2L18.1 6.7C18.3 6.5 18.3 6.3 18.2 6.1L16.4 3C16.3 2.8 16.1 2.7 15.9 2.8L13.7 3.7C13.2 3.3 12.7 3 12.1 2.8L11.8 0.4C11.7 0.2 11.5 0 11.3 0H7.7C7.5 0 7.3 0.2 7.2 0.4L6.9 2.8C6.3 3 5.8 3.3 5.3 3.7L3.1 2.8C2.9 2.7 2.7 2.8 2.6 3L0.8 6.1C0.7 6.3 0.7 6.5 0.9 6.7L2.8 8.2C2.7 8.8 2.6 9.4 2.6 10C2.6 10.6 2.7 11.2 2.8 11.8L0.9 13.3C0.7 13.5 0.7 13.7 0.8 13.9L2.6 17C2.7 17.2 2.9 17.3 3.1 17.2L5.3 16.3C5.8 16.7 6.3 17 6.9 17.2L7.2 19.6C7.3 19.8 7.5 20 7.7 20H11.3C11.5 20 11.7 19.8 11.8 19.6L12.1 17.2C12.7 17 13.2 16.7 13.7 16.3L15.9 17.2C16.1 17.3 16.3 17.2 16.4 17L18.2 13.9C18.3 13.7 18.3 13.5 18.1 13.3L16.2 11.8ZM9.5 13.5C7.6 13.5 6 11.9 6 10C6 8.1 7.6 6.5 9.5 6.5C11.4 6.5 13 8.1 13 10C13 11.9 11.4 13.5 9.5 13.5Z"
            fill={active ? "#00685F" : "#3D4947"}
          />
        </svg>
      ),
    },
  ];

  return (
    <aside
      className={`bg-white w-[256px] shrink-0 border-r border-[#bcc9c6]/40 flex flex-col justify-between select-none ${className}`}
      data-name="SideNavBar"
    >
      <div className="flex flex-col">
        {/* Logo & Portal Header */}
        <div className="pt-8 pb-6 px-6">
          <div className="flex items-center gap-3">
            <div className="relative size-10 rounded-full border border-[#bcc9c6] overflow-hidden bg-[#e1e8fd] shrink-0">
              <img
                src={imgOrganizerAvatar}
                alt="EventHub Portal"
                className="size-full object-cover"
              />
            </div>
            <div className="flex flex-col">
              <span className="font-['Inter'] font-bold text-[22px] tracking-[-0.6px] text-[#00685f] leading-tight">
                EventHub
              </span>
              <span className="font-['Inter'] text-[13px] text-[#3d4947] font-medium">
                Organizer Portal
              </span>
            </div>
          </div>
        </div>

        {/* Navigation List */}
        <nav className="flex flex-col gap-1 px-3">
          {navItems.map((item) => {
            const active = item.isActive
              ? item.isActive(location.pathname)
              : item.exact
              ? location.pathname === item.path
              : location.pathname.startsWith(item.path);

            return (
              <Link
                key={item.label}
                to={item.path}
                className={`flex items-center gap-3 px-4 py-3 rounded-lg text-[14px] transition-all duration-150 ${
                  active
                    ? "bg-[#008378]/10 text-[#00685f] font-bold border-l-4 border-[#00685f] rounded-l-none"
                    : "text-[#3d4947] font-semibold hover:bg-gray-100/70"
                }`}
              >
                <div className="size-[20px] flex items-center justify-center shrink-0">
                  {item.icon(active)}
                </div>
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>
      </div>

      {/* Organizer Profile Card at Bottom */}
      <div className="p-4 border-t border-[#bcc9c6]/30">
        <div className="flex items-center gap-3 p-2 rounded-xl bg-[#f9f9ff] border border-[#bcc9c6]/40">
          <div className="size-9 rounded-full overflow-hidden shrink-0 border border-[#bcc9c6]">
            <img
              src={imgUserAvatar}
              alt="Organizer"
              className="size-full object-cover"
            />
          </div>
          <div className="flex flex-col min-w-0">
            <span className="text-[13px] font-semibold text-[#141b2b] truncate">
              {user?.name || "Event Masters"}
            </span>
            <span className="text-[11px] text-[#565e74] truncate">
              Verified Organizer
            </span>
          </div>
        </div>
      </div>
    </aside>
  );
}
