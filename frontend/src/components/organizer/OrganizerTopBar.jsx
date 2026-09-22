import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ORGANIZER_ROUTES } from "../../constants/eventConstants";
import { LogoutIcon } from "../layout/icons";
import { logout } from "../../api/authApi";
import imgUserAvatar from "../../assets/organizer/77afb3678a691019904dbffccf7db02c242ed0f3.png";
import useAuthStore from "../../store/authStore";
import useEventCreationStore from "../../store/eventCreationStore";
import useOrganizerStore from "../../store/organizerStore";

export default function OrganizerTopBar({ showCreateButton = true, onMenuToggle }) {
  const navigate = useNavigate();
  // const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const resetEventForm = useEventCreationStore((state) => state.resetForm);
  const profile = useOrganizerStore((state) => state.profile);

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

  return (
    <header className="h-16 bg-white border-b border-[#bcc9c6]/40 px-3 sm:px-6 flex items-center justify-between shrink-0 sticky top-0 z-30 gap-2 sm:gap-4">
      {/* Left: Mobile hamburger & Search Input */}
      <div className="flex items-center gap-2 flex-1 max-w-md">
        {onMenuToggle && (
          <button
            type="button"
            onClick={onMenuToggle}
            className="lg:hidden p-1.5 text-[#3d4947] hover:bg-gray-100 rounded-lg transition-colors shrink-0"
            aria-label="Open sidebar"
          >
            <svg className="size-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        )}
        <div className="flex items-center gap-2 bg-[#f9f9ff] border border-[#bcc9c6]/60 rounded-lg px-2.5 sm:px-3 py-1.5 w-full focus-within:border-[#00685f] focus-within:ring-1 focus-within:ring-[#00685f]/20 transition-all">
          <svg
            className="size-4 text-[#565e74] shrink-0"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
          <input
            type="text"
            placeholder="Search events, orders..."
            className="bg-transparent border-none text-[12px] sm:text-[13px] text-[#141b2b] placeholder-[#565e74] focus:outline-none w-full"
          />
        </div>
      </div>

      {/* Right Side Actions */}
      <div className="flex items-center gap-1.5 sm:gap-4 shrink-0">
        {/* Notification Bell */}
        <button
          type="button"
          aria-label="Notifications"
          className="relative p-1.5 sm:p-2 text-[#3d4947] hover:bg-gray-100 rounded-lg transition-colors"
        >
          <svg
            className="size-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="1.8"
              d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
            />
          </svg>
          <span className="absolute top-1.5 right-1.5 size-2 bg-[#ba1a1a] rounded-full" />
        </button>

        {/* User Profile Thumbnail */}
        <Link
          to={ORGANIZER_ROUTES.PROFILE}
          className="flex items-center gap-2 p-1 sm:p-1.5 rounded-lg hover:bg-gray-100/80 transition-colors cursor-pointer group"
          title="View Organizer Profile"
        >
          <div className="size-7 sm:size-8 rounded-full border border-[#bcc9c6] overflow-hidden shrink-0">
            <img
              src={profile?.logo_url || imgUserAvatar}
              alt="Profile"
              className="size-full object-cover"
            />
          </div>
          <span className="text-[13px] font-medium text-[#141b2b] group-hover:text-[#00685f] transition-colors hidden md:inline-block truncate max-w-[120px]">
            {profile?.business_name || profile?.email || "Event Masters"}
          </span>
        </Link>

        {/* Logout Action */}
        <button
          type="button"
          onClick={handleLogout}
          disabled={isLoggingOut}
          className="flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 text-xs font-semibold text-[#ba1a1a] hover:bg-[#ffdad6]/60 border border-[#bcc9c6]/40 hover:border-[#ba1a1a]/30 rounded-lg transition-colors cursor-pointer disabled:opacity-50"
          title="Sign out of Organizer Portal"
        >
          <LogoutIcon className="w-3.5 h-3.5 text-[#ba1a1a]" />
          <span className="hidden sm:inline">{isLoggingOut ? "..." : "Logout"}</span>
        </button>

        {/* Create Event Primary CTA */}
        {showCreateButton && (
          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
            className="flex items-center gap-1.5 sm:gap-2 bg-[#00685f] hover:bg-[#005550] text-white font-semibold text-xs sm:text-[14px] px-2.5 sm:px-4 py-1.5 sm:py-2 rounded-lg shadow-sm transition-colors cursor-pointer shrink-0"
          >
            <svg
              className="size-3.5 sm:size-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2.2"
                d="M12 4v16m8-8H4"
              />
            </svg>
            <span className="hidden sm:inline">Create Event</span>
            <span className="sm:hidden">Create</span>
          </button>
        )}
      </div>
    </header>
  );
}
