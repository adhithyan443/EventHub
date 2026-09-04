import { Link, useLocation } from "react-router-dom";

const navItems = [
  {
    path: "/admin/dashboard",
    label: "Dashboard",
    icon: (
      <svg width="15" height="15" viewBox="0 0 15 15" fill="none">
        <path
          d="M1.5 0.5C0.948 0.5 0.5 0.948 0.5 1.5V6.5C0.5 7.052 0.948 7.5 1.5 7.5H6.5C7.052 7.5 7.5 7.052 7.5 6.5V1.5C7.5 0.948 7.052 0.5 6.5 0.5H1.5ZM8.5 0.5C7.948 0.5 7.5 0.948 7.5 1.5V6.5C7.5 7.052 7.948 7.5 8.5 7.5H13.5C14.052 7.5 14.5 7.052 14.5 6.5V1.5C14.5 0.948 14.052 0.5 13.5 0.5H8.5ZM1.5 8.5C0.948 8.5 0.5 8.948 0.5 9.5V13.5C0.5 14.052 0.948 14.5 1.5 14.5H6.5C7.052 14.5 7.5 14.052 7.5 13.5V9.5C7.5 8.948 7.052 8.5 6.5 8.5H1.5ZM8.5 8.5C7.948 8.5 7.5 8.948 7.5 9.5V13.5C7.5 14.052 7.948 14.5 8.5 14.5H13.5C14.052 14.5 14.5 14.052 14.5 13.5V9.5C14.5 8.948 14.052 8.5 13.5 8.5H8.5Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/users",
    label: "Users",
    icon: (
      <svg width="18" height="13" viewBox="0 0 18 13" fill="none">
        <path
          d="M13 6C14.66 6 15.99 4.66 15.99 3C15.99 1.34 14.66 0 13 0C11.34 0 10 1.34 10 3C10 4.66 11.34 6 13 6ZM5 6C6.66 6 7.99 4.66 7.99 3C7.99 1.34 6.66 0 5 0C3.34 0 2 1.34 2 3C2 4.66 3.34 6 5 6ZM5 8C3.34 8 0 8.84 0 10.5V12.33H10V10.5C10 8.84 6.66 8 5 8ZM13 8C12.79 8 12.55 8.01 12.3 8.03C13.19 8.68 13.83 9.55 13.83 10.5V12.33H18V10.5C18 8.84 14.66 8 13 8Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/applications",
    label: "Organizer Applications",
    icon: (
      <svg width="17" height="14" viewBox="0 0 17 14" fill="none">
        <path
          d="M15.17 0H1.83C0.82 0 0 0.9 0 2V12C0 13.1 0.82 14 1.83 14H15.17C16.18 14 17 13.1 17 12V2C17 0.9 16.18 0 15.17 0ZM15.17 12H1.83V4H15.17V12ZM1.83 2H15.17V3H1.83V2ZM4 6H9V7.5H4V6ZM4 8.5H9V10H4V8.5ZM11 6H13V10H11V6Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/organizers",
    label: "Organizers",
    icon: (
      <svg width="17" height="16" viewBox="0 0 17 16" fill="none">
        <path
          d="M8.5 0C6.57 0 5 1.57 5 3.5C5 5.43 6.57 7 8.5 7C10.43 7 12 5.43 12 3.5C12 1.57 10.43 0 8.5 0ZM8.5 9C5.83 9 0.5 10.34 0.5 13V15H16.5V13C16.5 10.34 11.17 9 8.5 9Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/events",
    label: "Events",
    icon: (
      <svg width="15" height="17" viewBox="0 0 15 17" fill="none">
        <path
          d="M13.5 1.5H12.75V0H11.25V1.5H3.75V0H2.25V1.5H1.5C0.675 1.5 0 2.175 0 3V15C0 15.825 0.675 16.5 1.5 16.5H13.5C14.325 16.5 15 15.825 15 15V3C15 2.175 14.325 1.5 13.5 1.5ZM13.5 15H1.5V5.25H13.5V15ZM1.5 3.75H13.5V5.25H1.5V3.75ZM4.5 8.25H7.5V11.25H4.5V8.25Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/bookings",
    label: "Bookings",
    icon: (
      <svg width="18" height="13" viewBox="0 0 18 13" fill="none">
        <path
          d="M16.5 0H1.5C0.675 0 0.0075 0.675 0.0075 1.5L0 10.5C0 11.325 0.675 12 1.5 12H16.5C17.325 12 18 11.325 18 10.5V1.5C18 0.675 17.325 0 16.5 0ZM16.5 10.5H1.5V6H16.5V10.5ZM16.5 3H1.5V1.5H16.5V3Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/payments",
    label: "Payments & Refunds",
    icon: (
      <svg width="18" height="13" viewBox="0 0 18 13" fill="none">
        <path
          d="M16.5 0H1.5C0.675 0 0 0.675 0 1.5V11.5C0 12.325 0.675 13 1.5 13H16.5C17.325 13 18 12.325 18 11.5V1.5C18 0.675 17.325 0 16.5 0ZM16.5 11.5H1.5V7H16.5V11.5ZM16.5 4H1.5V1.5H16.5V4Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/categories",
    label: "Categories",
    icon: (
      <svg width="16" height="17" viewBox="0 0 16 17" fill="none">
        <path
          d="M1.5 7H6.5C7.33 7 8 6.33 8 5.5V1.5C8 0.67 7.33 0 6.5 0H1.5C0.67 0 0 0.67 0 1.5V5.5C0 6.33 0.67 7 1.5 7ZM9.5 7H14.5C15.33 7 16 6.33 16 5.5V1.5C16 0.67 15.33 0 14.5 0H9.5C8.67 0 8 0.67 8 1.5V5.5C8 6.33 8.67 7 9.5 7ZM1.5 17H6.5C7.33 17 8 16.33 8 15.5V11.5C8 10.67 7.33 10 6.5 10H1.5C0.67 10 0 10.67 0 11.5V15.5C0 16.33 0.67 17 1.5 17ZM9.5 17H14.5C15.33 17 16 16.33 16 15.5V11.5C16 10.67 15.33 10 14.5 10H9.5C8.67 10 8 10.67 8 11.5V15.5C8 16.33 8.67 17 9.5 17Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
  {
    path: "/admin/reports",
    label: "Reports",
    icon: (
      <svg width="15" height="15" viewBox="0 0 15 15" fill="none">
        <path
          d="M13.5 0H1.5C0.675 0 0 0.675 0 1.5V13.5C0 14.325 0.675 15 1.5 15H13.5C14.325 15 15 14.325 15 13.5V1.5C15 0.675 14.325 0 13.5 0ZM6 11.25L2.25 7.5L3.3075 6.4425L6 9.1275L11.6925 3.435L12.75 4.5L6 11.25Z"
          fill="currentColor"
        />
      </svg>
    ),
  },
];

export default function AdminSidebar() {
  const location = useLocation();

  return (
    <aside className="bg-[#f9f9ff] border-r border-[#bcc9c6] flex flex-col h-full w-[239px] shrink-0 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
      {/* Brand Header */}
      <div className="px-4 pt-4 pb-6">
        <Link to="/admin/dashboard" className="block">
          <div className="text-[#00685f] text-2xl font-bold leading-8 tracking-tight">
            EventHub
          </div>
          <div className="text-[#3d4947] text-xs font-medium tracking-[0.6px] mt-1">
            Admin Console
          </div>
        </Link>
      </div>

      {/* Nav items */}
      <nav className="flex-1 overflow-y-auto px-2 flex flex-col gap-1 pb-4">
        {navItems.map(({ path, label, icon }) => {
          const active =
            location.pathname === path ||
            (path === "/admin/dashboard" && location.pathname === "/admin");

          return (
            <Link
              key={path}
              to={path}
              className={`flex items-center gap-4 w-full px-4 py-2 rounded-lg text-left transition-colors ${
                active
                  ? "bg-[#dae2fd] text-[#00685f]"
                  : "text-[#3d4947] hover:bg-[#eef0fb]"
              }`}
            >
              <span className={active ? "text-[#00685f]" : "text-[#3d4947]"}>
                {icon}
              </span>
              <span
                className={`text-sm font-semibold ${
                  active ? "text-[#00685f]" : "text-[#3d4947]"
                }`}
              >
                {label}
              </span>
            </Link>
          );
        })}
      </nav>

      {/* Create Event CTA */}
      {/* <div className="px-4 py-4 border-t border-[#dce2f7]">
        <button
          type="button"
          className="bg-[#00685f] text-white text-sm font-semibold w-full flex items-center justify-center gap-2 py-2 rounded-lg hover:bg-[#005a52] transition-colors cursor-pointer"
        >
          <svg width="11" height="11" viewBox="0 0 11 11" fill="none">
            <path
              d="M10.5 4.5H6.5V0.5H4.5V4.5H0.5V6.5H4.5V10.5H6.5V6.5H10.5V4.5Z"
              fill="white"
            />
          </svg>
          Create Event
        </button>
      </div> */}
    </aside>
  );
}
