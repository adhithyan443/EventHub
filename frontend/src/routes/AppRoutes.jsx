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
import ProtectedRoute from "../components/auth/ProtectedRoute";
import AdminLayout from "../components/admin/AdminLayout";
import AdminDashboardPage from "../pages/admin/AdminDashboardPage";
import AdminApplicationsPage from "../pages/admin/AdminApplicationsPage";
import AdminOrganizersPage from "../pages/admin/AdminOrganizersPage";
import AdminPlaceholderPage from "../pages/admin/AdminPlaceholderPage";

// Organizer UI Components
import OrganizerLayout from "../components/organizer/OrganizerLayout";
import OrganizerDashboardPage from "../pages/organizer/OrganizerDashboardPage";
import CreateEventStep1BasicInfoPage from "../pages/organizer/create-event/CreateEventStep1BasicInfoPage";
import CreateEventStep2DetailsPage from "../pages/organizer/create-event/CreateEventStep2DetailsPage";
import CreateEventStep3TicketSelectionPage from "../pages/organizer/create-event/CreateEventStep3TicketSelectionPage";
import CreateEventTicketTypesPage from "../pages/organizer/create-event/CreateEventTicketTypesPage";
import CreateEventSeatConfigurationPage from "../pages/organizer/create-event/CreateEventSeatConfigurationPage";
import CreateEventReviewPublishPage from "../pages/organizer/create-event/CreateEventReviewPublishPage";
import OrganizerPlaceholderPage from "../pages/organizer/OrganizerPlaceholderPage";

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

            {/* Default application entry route redirects to /login */}
            <Route path="/" element={<Navigate to="/login" replace />} />

            {/* Preserved User Index / Events Discovery Routes */}
            <Route path="/events" element={<DiscoverEventsPage />} />
            <Route path="/users" element={<DiscoverEventsPage />} />
            <Route path="/profile" element={<ProfilePage />} />

            {/* Become an Organizer Workflow */}
            <Route path="/become-organizer" element={<BecomeOrganizerPage />} />
            <Route path="/become-organizer/status" element={<OrganizerStatusPage />} />

            {/* Protected Admin Console */}
            <Route element={<ProtectedRoute allowedRoles={["ADMIN"]} />}>
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
            </Route>

            {/* Organizer Portal */}
            {/* Note: In this UI-only phase, accessible directly for UI review. Structured to allow wrapping in ProtectedRoute allowedRoles={["ORGANIZER"]} later. */}
            <Route path="/organizer" element={<OrganizerLayout />}>
                <Route index element={<Navigate to="/organizer/dashboard" replace />} />
                <Route path="dashboard" element={<OrganizerDashboardPage />} />

                {/* Multi-Step Event Creation Flow */}
                <Route path="events/create" element={<Navigate to="/organizer/events/create/step-1" replace />} />
                <Route path="events/create/step-1" element={<CreateEventStep1BasicInfoPage />} />
                <Route path="events/create/step-2" element={<CreateEventStep2DetailsPage />} />
                <Route path="events/create/step-3" element={<CreateEventStep3TicketSelectionPage />} />
                <Route path="events/create/ticket-types" element={<CreateEventTicketTypesPage />} />
                <Route path="events/create/seat-configuration" element={<CreateEventSeatConfigurationPage />} />
                <Route path="events/create/review" element={<CreateEventReviewPublishPage />} />

                {/* Organizer Placeholders */}
                <Route path="events" element={<OrganizerPlaceholderPage title="My Events" />} />
                <Route path="bookings" element={<OrganizerPlaceholderPage title="Bookings" />} />
                <Route path="attendees" element={<OrganizerPlaceholderPage title="Attendees" />} />
                <Route path="reports" element={<OrganizerPlaceholderPage title="Reports" />} />
                <Route path="settings" element={<OrganizerPlaceholderPage title="Settings" />} />
            </Route>

            {/* Redirect alias: /organizer/create-event -> /organizer/events/create/step-1 */}
            <Route path="/organizer/create-event" element={<Navigate to="/organizer/events/create/step-1" replace />} />
        </Routes>
    );
}