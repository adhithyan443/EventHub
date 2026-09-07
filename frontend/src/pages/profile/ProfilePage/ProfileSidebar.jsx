import { NavLink, useNavigate } from "react-router-dom";
import { UserIcon, TicketIcon, HeartIcon, SettingsIcon } from "../../../components/layout/icons";

const navItems = [
    { label: "Profile", to: "/profile", icon: UserIcon },
    { label: "Tickets", to: "/my-tickets", icon: TicketIcon },
    { label: "Favorites", to: "/favorites", icon: HeartIcon },
    { label: "Settings", to: "/settings", icon: SettingsIcon },
];

export default function ProfileSidebar({ user }) {
    const avatarUrl = user?.avatar || "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=150&auto=format&fit=crop&q=80";
    const navigate = useNavigate();

    return (
        <aside className="w-64 shrink-0 bg-[#f0f4fa] border border-slate-200/70 rounded-2xl p-5 flex flex-col justify-between self-stretch">
            <div className="flex flex-col gap-6">
                <div className="flex items-center gap-3">
                    <img
                        src={avatarUrl}
                        alt={user.fullName}
                        className="h-11 w-11 rounded-full object-cover shrink-0 border border-slate-200"
                    />
                    <div className="min-w-0">
                        <p className="font-bold text-slate-900 text-sm truncate">{user.fullName}</p>
                        <p className="text-xs text-slate-500">Member since {user.memberSince}</p>
                    </div>
                </div>

                <nav className="flex flex-col gap-1.5">
                    {navItems.map(({ label, to, icon: Icon }) => (
                        <NavLink
                            key={to}
                            to={to}
                            end={to === "/profile"}
                            className={({ isActive }) =>
                                `flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-colors ${isActive
                                    ? "bg-primary text-white font-semibold shadow-sm"
                                    : "text-slate-600 hover:bg-white/80 hover:text-slate-900"
                                }`
                            }
                        >
                            <Icon />
                            {label}
                        </NavLink>
                    ))}
                </nav>
            </div>

            <div className="pt-8">
                {user?.role === "ORGANIZER" ? (
                    <div className="w-full py-2.5 px-3 rounded-lg bg-emerald-100 text-emerald-800 text-xs font-bold text-center border border-emerald-200 flex items-center justify-center gap-1.5 shadow-xs">
                        <span className="w-2 h-2 rounded-full bg-emerald-500" />
                        Active Organizer
                    </div>
                ) : (
                    <button
                        onClick={() => navigate("/become-organizer")}
                        className="w-full h-10 rounded-lg bg-[#007066] hover:bg-[#005c54] text-white text-sm font-semibold transition-colors cursor-pointer shadow-xs">
                        Become an Organizer
                    </button>
                )}
            </div>
        </aside>
    );
}