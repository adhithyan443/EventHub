import { useState } from "react";

export default function AdminTopBar() {
  const [searchValue, setSearchValue] = useState("");

  return (
    <header className="bg-white border-b border-[#bcc9c6] h-16 flex items-center justify-between px-6 shrink-0 shadow-[0px_1px_1px_rgba(0,0,0,0.05)] z-10">
      {/* Search */}
      <div className="relative w-64 md:w-80">
        <svg
          className="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none"
          width="18"
          height="18"
          viewBox="0 0 18 18"
          fill="none"
        >
          <path
            d="M11.5 10H10.71L10.43 9.73C11.41 8.59 12 7.11 12 5.5C12 1.91 9.09 -1 5.5 -1C1.91 -1 -1 1.91 -1 5.5C-1 9.09 1.91 12 5.5 12C7.11 12 8.59 11.41 9.73 10.43L10 10.71V11.5L15 16.49L16.49 15L11.5 10ZM5.5 10C3.01 10 1 7.99 1 5.5C1 3.01 3.01 1 5.5 1C7.99 1 10 3.01 10 5.5C10 7.99 7.99 10 5.5 10Z"
            fill="#3D4947"
          />
        </svg>
        <input
          type="text"
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          placeholder="Search globally..."
          className="bg-[#f1f3ff] border border-[#bcc9c6] rounded-lg pl-10 pr-4 py-2 text-sm text-[#141b2b] placeholder:text-[#6b7280] w-full focus:outline-none focus:ring-2 focus:ring-[#00685f]/30 focus:border-[#00685f] transition-all"
        />
      </div>

      {/* Right actions */}
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-3">
          {/* Bell Notifications */}
          <button
            type="button"
            title="Notifications"
            className="p-2 rounded-full hover:bg-[#f1f3ff] text-[#3D4947] transition-colors relative cursor-pointer"
          >
            <svg width="16" height="20" viewBox="0 0 16 20" fill="none">
              <path
                d="M8 20C9.1 20 10 19.1 10 18H6C6 19.1 6.9 20 8 20ZM14 14V9C14 5.93 12.37 3.36 9.5 2.68V2C9.5 1.17 8.83 0.5 8 0.5C7.17 0.5 6.5 1.17 6.5 2V2.68C3.64 3.36 2 5.92 2 9V14L0 16V17H16V16L14 14Z"
                fill="currentColor"
              />
            </svg>
            <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-[#ba1a1a] rounded-full" />
          </button>

          {/* Settings */}
          <button
            type="button"
            title="Settings"
            className="p-2 rounded-full hover:bg-[#f1f3ff] text-[#3D4947] transition-colors cursor-pointer"
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path
                d="M17.14 10.94C17.18 10.64 17.2 10.33 17.2 10C17.2 9.68 17.18 9.36 17.13 9.06L19.16 7.48C19.34 7.34 19.39 7.07 19.28 6.87L17.36 3.55C17.24 3.33 16.99 3.26 16.77 3.33L14.38 4.29C13.88 3.91 13.35 3.59 12.76 3.35L12.4 0.81C12.36 0.57 12.16 0.4 11.92 0.4H8.08C7.84 0.4 7.65 0.57 7.61 0.81L7.25 3.35C6.66 3.59 6.12 3.92 5.63 4.29L3.24 3.33C3.02 3.25 2.77 3.33 2.65 3.55L0.74 6.87C0.62 7.08 0.66 7.34 0.86 7.48L2.89 9.06C2.84 9.36 2.8 9.69 2.8 10C2.8 10.31 2.82 10.64 2.87 10.94L0.84 12.52C0.66 12.66 0.61 12.93 0.72 13.13L2.64 16.45C2.76 16.67 3.01 16.74 3.23 16.67L5.62 15.71C6.12 16.09 6.65 16.41 7.24 16.65L7.6 19.19C7.65 19.43 7.84 19.6 8.08 19.6H11.92C12.16 19.6 12.36 19.43 12.39 19.19L12.75 16.65C13.34 16.41 13.88 16.09 14.37 15.71L16.76 16.67C16.98 16.75 17.23 16.67 17.35 16.45L19.27 13.13C19.39 12.91 19.34 12.66 19.15 12.52L17.14 10.94ZM10 13.6C8.02 13.6 6.4 11.98 6.4 10C6.4 8.02 8.02 6.4 10 6.4C11.98 6.4 13.6 8.02 13.6 10C13.6 11.98 11.98 13.6 10 13.6Z"
                fill="currentColor"
              />
            </svg>
          </button>
        </div>

        <div className="w-px h-8 bg-[#bcc9c6]" />

        {/* Admin Avatar */}
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-[#e8eaf6] border border-[#bcc9c6] flex items-center justify-center text-[#5c6bc0] font-bold text-sm select-none shadow-sm">
            A
          </div>
          <div className="hidden sm:block text-left">
            <div className="text-[#141b2b] text-xs font-semibold">Admin User</div>
            <div className="text-[#6d7a77] text-[10px]">Super Administrator</div>
          </div>
        </div>
      </div>
    </header>
  );
}
