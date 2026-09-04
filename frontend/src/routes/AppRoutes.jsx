import { Routes, Route, Navigate } from "react-router-dom";

import LoginPage from "../pages/auth/LoginPage";
import RegisterPage from "../pages/auth/RegisterPage";
import OtpPage from "../pages/auth/OtpPage";
import OAuthCallbackPage from "../pages/auth/OAuthCallbackPage";
import ForgotPasswordPage from "../pages/auth/ForgotPasswordPage";
import ResetPasswordPage from "../pages/auth/ResetPasswordPage";
import DiscoverEventsPage from "../pages/events/DiscoverEventsPage";
import ProfilePage from "../pages/profile/ProfilePage";
import BecomeOrganizerPage from "../pages/organizer/BecomeOrganizerPage";
import OrganizerStatusPage from "../pages/organizer/OrganizerStatusPage";

// Admin UI Components
import AdminLayout from "../components/admin/AdminLayout";
import AdminDashboardPage from "../pages/admin/AdminDashboardPage";
import AdminApplicationsPage from "../pages/admin/AdminApplicationsPage";
import AdminOrganizersPage from "../pages/admin/AdminOrganizersPage";
import AdminPlaceholderPage from "../pages/admin/AdminPlaceholderPage";

export default function AppRoutes() {
    return (
        <Routes>
            {/* Authentication */}
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/verify-otp" element={<OtpPage />} />
            <Route path="/forgot-password" element={<ForgotPasswordPage />} />
            <Route path="/reset-password" element={<ResetPasswordPage />} />

            {/* Google OAuth callback */}
            <Route
                path="/oauth/callback"
                element={<OAuthCallbackPage />}
            />

            {/* Customer home */}
            <Route path="/" element={<DiscoverEventsPage />} />
            <Route path="/profile" element={<ProfilePage />} />

            {/* Become an Organizer Workflow */}
            <Route path="/become-organizer" element={<BecomeOrganizerPage />} />
            <Route path="/become-organizer/status" element={<OrganizerStatusPage />} />

            {/* Admin Console */}
            <Route path="/admin" element={<AdminLayout />}>
                <Route index element={<Navigate to="/admin/dashboard" replace />} />
                <Route path="dashboard" element={<AdminDashboardPage />} />
                <Route path="applications" element={<AdminApplicationsPage />} />
                <Route path="organizers" element={<AdminOrganizersPage />} />
                <Route path="users" element={<AdminPlaceholderPage title="Users" />} />
                <Route path="events" element={<AdminPlaceholderPage title="Events" />} />
                <Route path="bookings" element={<AdminPlaceholderPage title="Bookings" />} />
                <Route path="payments" element={<AdminPlaceholderPage title="Payments & Refunds" />} />
                <Route path="categories" element={<AdminPlaceholderPage title="Categories" />} />
                <Route path="reports" element={<AdminPlaceholderPage title="Reports" />} />
            </Route>
        </Routes>
    );
}