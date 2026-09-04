import { Link, NavLink } from "react-router-dom";
import { SearchIcon, BellIcon } from "./icons";

const navLinks = [
  { label: "Events", to: "/" },
  { label: "Categories", to: "/categories" },
  { label: "My Tickets", to: "/my-tickets" },
];

export default function AppHeader() {
  return (
    <header className="sticky top-0 z-10 bg-background/95 backdrop-blur-sm border-b border-border">
      <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
        <div className="flex items-center gap-8">
          <Link to="/" className="font-display text-2xl font-bold text-primary">
            EventHub
          </Link>

          <div className="hidden md:flex items-center gap-3 bg-[#f0f4fa] border border-slate-200 rounded-full px-4 py-1.5 w-64">
            <span className="text-slate-400"><SearchIcon /></span>
            <input
              type="text"
              placeholder="Search events..."
              className="bg-transparent text-sm text-slate-800 placeholder:text-slate-400 focus:outline-none w-full"
            />
          </div>

          <nav className="hidden lg:flex items-center gap-6">
            {navLinks.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end={link.to === "/"}
                className={({ isActive }) =>
                  `text-sm py-1.5 border-b-2 transition-colors ${
                    isActive
                      ? "border-primary text-primary font-bold"
                      : "border-transparent text-ink/70 hover:text-ink"
                  }`
                }
              >
                {link.label}
              </NavLink>
            ))}
          </nav>
        </div>

        <div className="flex items-center gap-4">
          <button className="text-slate-600 hover:text-slate-900" aria-label="Notifications">
            <BellIcon />
          </button>
          <Link to="/profile" className="h-8 w-8 rounded-full overflow-hidden block border border-slate-200 shrink-0" aria-label="View profile">
            <img
              src="https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=150&auto=format&fit=crop&q=80"
              alt="Profile"
              className="h-full w-full object-cover"
            />
          </Link>
        </div>
      </div>
    </header>
  );
}