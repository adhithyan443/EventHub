import { useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { SearchIcon, BellIcon } from "./icons";

const navLinks = [
  { label: "Events", to: "/events" },
  { label: "Categories", to: "/categories" },
  { label: "My Tickets", to: "/my-tickets" },
];

export default function AppHeader() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <header className="sticky top-0 z-20 bg-background/95 backdrop-blur-sm border-b border-border">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
        <div className="flex items-center gap-4 sm:gap-8">
          {/* Mobile hamburger menu toggle */}
          <button
            type="button"
            onClick={() => setMobileMenuOpen((prev) => !prev)}
            className="lg:hidden p-2 text-slate-600 hover:text-slate-900 rounded-lg hover:bg-slate-100 transition-colors"
            aria-label="Toggle navigation menu"
          >
            {mobileMenuOpen ? (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            )}
          </button>

          <Link to="/events" className="font-display text-xl sm:text-2xl font-bold text-primary">
            EventHub
          </Link>

          <div className="hidden md:flex items-center gap-3 bg-[#f0f4fa] border border-slate-200 rounded-full px-4 py-1.5 w-56 lg:w-64">
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
                end={link.to === "/events"}
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

        <div className="flex items-center gap-3 sm:gap-4">
          <button className="text-slate-600 hover:text-slate-900 p-1" aria-label="Notifications">
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

      {/* Mobile navigation drawer/dropdown */}
      {mobileMenuOpen && (
        <div className="lg:hidden border-t border-border bg-white px-4 py-3 shadow-md">
          <div className="flex md:hidden items-center gap-2 bg-[#f0f4fa] border border-slate-200 rounded-lg px-3 py-2 mb-3">
            <span className="text-slate-400"><SearchIcon /></span>
            <input
              type="text"
              placeholder="Search events..."
              className="bg-transparent text-sm text-slate-800 placeholder:text-slate-400 focus:outline-none w-full"
            />
          </div>
          <nav className="flex flex-col gap-1">
            {navLinks.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end={link.to === "/events"}
                onClick={() => setMobileMenuOpen(false)}
                className={({ isActive }) =>
                  `text-sm px-3 py-2 rounded-lg transition-colors font-medium ${
                    isActive
                      ? "bg-primary/10 text-primary font-bold"
                      : "text-ink/80 hover:bg-slate-50 hover:text-ink"
                  }`
                }
              >
                {link.label}
              </NavLink>
            ))}
          </nav>
        </div>
      )}
    </header>
  );
}