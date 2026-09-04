import { useNavigate } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import ProfileSidebar from "./ProfileSidebar";
import PersonalInfoCard from "./PersonalInfoCard";
import AccountSettingsCard from "./AccountSettingsCard";
import ActivityCard from "./ActivityCard";
import OrganizerPromoCard from "./OrganizerPromoCard";
import { TicketIcon, CalendarIcon, RefreshIcon } from "../../../components/layout/icons";
import { logout } from "../../../api/authApi";
import useAuthStore from "../../../store/authStore";

export default function ProfilePage() {
    const navigate = useNavigate();
    const user = useAuthStore((state) => state.user);
    const clearAuth = useAuthStore((state) => state.clearAuth);

    // TODO: replace with real fields once /auth/profile returns phone/memberSince
    const displayUser = {
        fullName: user?.fullName || "Adhithyan R",
        email: user?.email || "customer@example.com",
        phone: user?.phone || "+91 XXXXX XXXXX",
        memberSince: "2023",
        avatar: user?.avatar || "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=150&auto=format&fit=crop&q=80",
    };

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
        <div className="min-h-screen bg-[#f8fafc] flex flex-col justify-between">
            <AppHeader />

            <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-8 flex flex-col lg:flex-row gap-6 items-stretch">
                <ProfileSidebar user={displayUser} />

                <div className="flex-1 grid grid-cols-1 lg:grid-cols-12 gap-6 w-full items-start">
                    <div className="lg:col-span-5 flex flex-col gap-6">
                        <PersonalInfoCard user={displayUser} />
                        <AccountSettingsCard onLogout={handleLogout} />
                    </div>

                    <div className="lg:col-span-7 flex flex-col gap-6">
                        <div>
                            <h2 className="font-display text-lg font-bold text-slate-900 mb-3">My Activity</h2>
                            <div className="grid grid-cols-2 gap-4">
                                <ActivityCard icon={CalendarIcon} title="My Bookings" description="View upcoming and past events." tone="primary" />
                                <ActivityCard icon={TicketIcon} title="My Tickets" description="Access your event passes." tone="soft" />
                                <ActivityCard icon={RefreshIcon} title="Refunds & Transactions" description="Track refund status and billing history." tone="soft" wide />
                            </div>
                        </div>

                        <div>
                            <h2 className="font-display text-lg font-bold text-slate-900 mb-3">Organizer Status</h2>
                            <OrganizerPromoCard />
                        </div>
                    </div>
                </div>
            </main>

            <Footer />
        </div>
    );
}