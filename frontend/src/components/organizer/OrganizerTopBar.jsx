import { useNavigate } from "react-router-dom";
import { ORGANIZER_ROUTES } from "../../constants/eventConstants";
import imgUserAvatar from "../../assets/organizer/77afb3678a691019904dbffccf7db02c242ed0f3.png";
import useAuthStore from "../../store/authStore";

export default function OrganizerTopBar({ showCreateButton = true }) {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);

  return (
    <header className="h-16 bg-white border-b border-[#bcc9c6]/40 px-6 flex items-center justify-between shrink-0 sticky top-0 z-30">
      {/* Search Input */}
      <div className="flex items-center gap-2 bg-[#f9f9ff] border border-[#bcc9c6]/60 rounded-lg px-3 py-1.5 w-full max-w-md focus-within:border-[#00685f] focus-within:ring-1 focus-within:ring-[#00685f]/20 transition-all">
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
          placeholder="Search events, orders, attendees..."
          className="bg-transparent border-none text-[13px] text-[#141b2b] placeholder-[#565e74] focus:outline-none w-full"
        />
      </div>

      {/* Right Side Actions */}
      <div className="flex items-center gap-4">
        {/* Notification Bell */}
        <button
          type="button"
          aria-label="Notifications"
          className="relative p-2 text-[#3d4947] hover:bg-gray-100 rounded-lg transition-colors"
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
        <div className="flex items-center gap-2">
          <div className="size-8 rounded-full border border-[#bcc9c6] overflow-hidden">
            <img
              src={imgUserAvatar}
              alt="Profile"
              className="size-full object-cover"
            />
          </div>
          <span className="text-[13px] font-medium text-[#141b2b] hidden md:inline-block">
            {user?.name || "Event Masters"}
          </span>
        </div>

        {/* Create Event Primary CTA */}
        {showCreateButton && (
          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
            className="flex items-center gap-2 bg-[#00685f] hover:bg-[#005550] text-white font-semibold text-[14px] px-4 py-2 rounded-lg shadow-sm transition-colors cursor-pointer"
          >
            <svg
              className="size-4"
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
            <span>Create Event</span>
          </button>
        )}
      </div>
    </header>
  );
}
