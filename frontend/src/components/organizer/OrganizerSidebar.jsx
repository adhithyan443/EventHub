import { useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { ORGANIZER_ROUTES } from "../../constants/eventConstants";
import { LogoutIcon } from "../layout/icons";
import { logout } from "../../api/authApi";
import imgUserAvatar from "../../assets/organizer/77afb3678a691019904dbffccf7db02c242ed0f3.png";
import useAuthStore from "../../store/authStore";
import useEventCreationStore from "../../store/eventCreationStore";
import useOrganizerStore from "../../store/organizerStore";

export default function OrganizerSidebar({ className = "", isOpen = false, onClose }) {
  const location = useLocation();
  const navigate = useNavigate();
  // const user = useAuthStore((state) => state.user);
  const profile = useOrganizerStore((state) => state.profile);

  const clearAuth = useAuthStore((state) => state.clearAuth);
  const resetEventForm = useEventCreationStore((state) => state.resetForm);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  async function handleLogout() {
    setIsLoggingOut(true);
    try {
      await logout();
    } catch (error) {
      console.error("Logout request failed:", error?.message || error);
    } finally {
      clearAuth();
      resetEventForm();
      setIsLoggingOut(false);
      navigate("/login", { replace: true });
    }
  }

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
      label: "Profile",
      path: ORGANIZER_ROUTES.PROFILE,
      icon: (active) => (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M10 10C12.21 10 14 8.21 14 6C14 3.79 12.21 2 10 2C7.79 2 6 3.79 6 6C6 8.21 7.79 10 10 10ZM10 12C7.33 12 2 13.34 2 16V18H18V16C18 13.34 12.67 12 10 12Z"
            fill={active ? "#00685F" : "#3D4947"}
          />
        </svg>
      ),
    },
  ];

  return (
    <>
      {/* Mobile Backdrop */}
      {isOpen && (
        <div
          onClick={onClose}
          className="fixed inset-0 bg-black/40 backdrop-blur-xs z-40 lg:hidden transition-opacity"
          aria-hidden="true"
        />
      )}

      <aside
        className={`fixed inset-y-0 left-0 z-50 lg:static lg:z-auto bg-white w-[256px] shrink-0 border-r border-[#bcc9c6]/40 flex flex-col justify-between select-none transition-transform duration-200 ease-in-out ${
          isOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        } ${className}`}
        data-name="SideNavBar"
      >
        <div className="flex flex-col">
          {/* Logo & Portal Header */}
          <div className="pt-6 pb-6 px-6 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="relative size-10 rounded-full border border-[#bcc9c6] overflow-hidden bg-[#e1e8fd] shrink-0">
                <img
                  src={profile?.logo_url || imgUserAvatar}
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
            {onClose && (
              <button
                type="button"
                onClick={onClose}
                className="lg:hidden p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 transition-colors"
                aria-label="Close sidebar"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
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
                  onClick={onClose}
                  className={`flex items-center gap-3 px-4 py-3 rounded-lg text-[14px] transition-all duration-150 ${active
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

        {/* Organizer Profile Card & Logout at Bottom */}
        <div className="p-4 border-t border-[#bcc9c6]/30 flex flex-col gap-2">
          <Link
            to={ORGANIZER_ROUTES.PROFILE}
            onClick={onClose}
            className="flex items-center gap-3 p-2 rounded-xl bg-[#f9f9ff] hover:bg-gray-100/80 border border-[#bcc9c6]/40 transition-colors cursor-pointer group"
            title="View Organizer Profile"
          >
            <div className="size-9 rounded-full overflow-hidden shrink-0 border border-[#bcc9c6]">
              <img
                src={profile?.logo_url || imgUserAvatar}
                alt="Organizer"
                className="size-full object-cover"
              />
            </div>
            <div className="flex flex-col min-w-0">
              <span className="text-[13px] font-semibold text-[#141b2b] group-hover:text-[#00685f] transition-colors truncate">
                {profile?.business_name || profile?.email || "Event Masters"}
              </span>
              <span className="text-[11px] text-[#565e74] truncate">
                Verified Organizer
              </span>
            </div>
          </Link>

          <button
            type="button"
            onClick={handleLogout}
            disabled={isLoggingOut}
            className="w-full flex items-center justify-center gap-2.5 px-4 py-2 rounded-lg text-sm font-semibold text-[#ba1a1a] hover:bg-[#ffdad6]/60 border border-[#bcc9c6]/40 hover:border-[#ba1a1a]/30 transition-colors cursor-pointer disabled:opacity-50"
            title="Sign out of Organizer Portal"
          >
            <LogoutIcon className="w-4 h-4 text-[#ba1a1a]" />
            <span>{isLoggingOut ? "Logging out..." : "Logout"}</span>
          </button>
        </div>
      </aside>
    </>
  );
}
