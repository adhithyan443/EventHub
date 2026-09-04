import { Routes, Route } from "react-router-dom";

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
        </Routes>
    );
}