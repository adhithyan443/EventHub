import { useState } from "react";
import { Link, NavLink, useNavigate } from "react-router-dom";
import { SearchIcon, BellIcon } from "./icons";
import { logout } from "../../api/authApi";
import useAuthStore from "../../store/authStore";

const navLinks = [
  { label: "Events", to: "/" },
  { label: "Categories", to: "/categories" },
  { label: "My Tickets", to: "/my-tickets" },
];

export default function AppHeader() {
  const [menuOpen, setMenuOpen] = useState(false);
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clearAuth);

  async function handleLogout() {
    try {
      await logout();
    } catch (error) {
      console.error("Logout failed:", error);
    } finally {
      clearAuth();
      navigate("/login", { replace: true });
    }
  }

  return (
    <header className="sticky top-0 z-10 bg-background/95 backdrop-blur-sm border-b border-border">
      <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
        <div className="flex items-center gap-8">
          <Link to="/" className="font-display text-2xl font-bold text-primary">
            EventHub
          </Link>

          <div className="hidden md:flex items-center gap-3 bg-surface border border-border rounded-full px-4 py-2 w-64">
            <span className="text-ink/50"><SearchIcon /></span>
            <input
              type="text"
              placeholder="Search events..."
              className="bg-transparent text-sm text-ink placeholder:text-ink/50 focus:outline-none w-full"
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
          <button className="text-ink/70 hover:text-ink" aria-label="Notifications">
            <BellIcon />
          </button>

          <div className="relative">
            <button
              onClick={() => setMenuOpen((open) => !open)}
              className="h-8 w-8 rounded-full bg-border overflow-hidden"
              aria-label="Account menu"
            />

            {menuOpen && (
              <div className="absolute right-0 mt-2 w-48 bg-white border border-border rounded-md shadow-lg py-1">
                {user && (
                  <p className="px-4 py-2 text-sm text-ink/60 border-b border-border truncate">
                    {user.fullName}
                  </p>
                )}
                <button
                  onClick={handleLogout}
                  className="w-full text-left px-4 py-2 text-sm text-ink hover:bg-background"
                >
                  Logout
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}