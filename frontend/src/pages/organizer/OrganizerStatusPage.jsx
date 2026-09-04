import { useState, useEffect } from "react";
import { Link, useNavigate } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import useOrganizerStore from "../../store/organizerStore";
import ApplicationDetailsModal from "./ApplicationDetailsModal";

export default function OrganizerStatusPage() {
  const navigate = useNavigate();
  const [isModalOpen, setIsModalOpen] = useState(false);

  const businessName = useOrganizerStore((state) => state.businessName) || "—";
  const businessType = useOrganizerStore((state) => state.businessType) || "—";
  const applicationId = useOrganizerStore((state) => state.applicationId);
  const applicationStatus = useOrganizerStore((state) => state.applicationStatus) || "PENDING";
  const rejectionReason = useOrganizerStore((state) => state.rejectionReason);
  const rawSubmissionDate = useOrganizerStore((state) => state.submissionDate);

  const isCheckingApplication = useOrganizerStore((state) => state.isCheckingApplication);
  const fetchApplication = useOrganizerStore((state) => state.fetchApplication);

  // Fetch current application from backend on mount
  useEffect(() => {
    fetchApplication();
  }, [fetchApplication]);

  // Format date cleanly
  const formattedDate = rawSubmissionDate
    ? (() => {
        try {
          const d = new Date(rawSubmissionDate);
          return isNaN(d.getTime())
            ? rawSubmissionDate
            : d.toLocaleDateString("en-US", {
                day: "numeric",
                month: "long",
                year: "numeric",
              });
        } catch {
          return rawSubmissionDate;
        }
      })()
    : "—";

  const isRejected = applicationStatus === "REJECTED";
  const isApproved = applicationStatus === "APPROVED";

  return (
    <div className="min-h-screen bg-[#F9F9FF] flex flex-col justify-between">
      <AppHeader />

      <main className="flex-1 max-w-4xl mx-auto w-full px-4 sm:px-6 py-8 sm:py-10">
        {/* Breadcrumb */}
        <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 mb-6 text-sm text-slate-500">
          <Link to="/profile" className="hover:text-primary transition-colors">
            Profile
          </Link>
          <svg width="6" height="10" viewBox="0 0 6 10" fill="none" aria-hidden="true">
            <path d="M1 1L5 5L1 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          <span className="font-semibold text-slate-900">Organizer Application</span>
        </nav>

        {/* Header */}
        <div className="mb-8">
          <h1 className="font-display text-2xl sm:text-3xl font-bold text-slate-900 tracking-tight mb-2">
            Organizer Application
          </h1>
          <p className="text-sm sm:text-base text-slate-600">
            Track the status of your organizer application.
          </p>
        </div>

        {/* Main Status Card */}
        <div className="bg-white rounded-2xl border border-slate-200/80 shadow-sm overflow-hidden mb-6 relative">
          {/* Subtle decorative background gradients */}
          <div className="absolute top-0 right-0 w-64 h-64 bg-primary/5 rounded-full blur-3xl -z-0 pointer-events-none" />
          <div className="absolute bottom-0 left-0 w-64 h-64 bg-indigo-500/5 rounded-full blur-3xl -z-0 pointer-events-none" />

          <div className="p-6 sm:p-8 relative z-10">
            {/* Top Row: App ID and Status badge */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-6 border-b border-slate-100 gap-4">
              <div>
                <span className="text-[11px] font-bold text-slate-400 tracking-wider uppercase">
                  Application ID
                </span>
                <p className="font-display text-base sm:text-xl font-bold text-slate-900 font-mono break-all">
                  {applicationId || "—"}
                </p>
              </div>

              <div className="flex sm:flex-col sm:items-end justify-between items-center gap-1">
                <span className="text-xs text-slate-500 font-medium">
                  {formattedDate}
                </span>
                <span
                  className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold tracking-wide ${
                    isRejected
                      ? "bg-red-50 text-red-600 border border-red-200"
                      : isApproved
                      ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                      : "bg-primary/10 text-primary border border-primary/20"
                  }`}
                >
                  <span
                    className={`w-1.5 h-1.5 rounded-full ${
                      isRejected
                        ? "bg-red-500"
                        : isApproved
                        ? "bg-emerald-500"
                        : "bg-primary animate-pulse"
                    }`}
                  />
                  {applicationStatus}
                </span>
              </div>
            </div>

            {/* Hero Message */}
            <div className="py-8 sm:py-10 flex flex-col items-center text-center max-w-lg mx-auto">
              <div
                className={`w-20 h-20 sm:w-24 sm:h-24 rounded-full flex items-center justify-center mb-6 shadow-xs ${
                  isRejected
                    ? "bg-red-50 text-red-600"
                    : isApproved
                    ? "bg-emerald-50 text-emerald-600"
                    : "bg-primary/10 text-primary"
                }`}
              >
                {isRejected ? (
                  <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <circle cx="12" cy="12" r="10" />
                    <line x1="15" y1="9" x2="9" y2="15" />
                    <line x1="9" y1="9" x2="15" y2="15" />
                  </svg>
                ) : isApproved ? (
                  <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                    <polyline points="22 4 12 14.01 9 11.01" />
                  </svg>
                ) : (
                  <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                    <circle cx="9" cy="7" r="4" />
                    <polyline points="16 11 18 13 22 9" />
                  </svg>
                )}
              </div>

              <h2 className="font-display text-xl sm:text-2xl font-bold text-slate-900 mb-2">
                {isRejected
                  ? "Application Rejected"
                  : isApproved
                  ? "Application Approved!"
                  : "Application Under Review"}
              </h2>
              <p className="text-sm sm:text-base font-semibold text-slate-800 mb-2">
                {isRejected
                  ? "Your organizer application requires corrections before it can be approved."
                  : isApproved
                  ? "Congratulations! You are now an active event organizer on EventHub."
                  : "Your organizer application has been submitted and is currently being reviewed."}
              </p>
              <p className="text-xs sm:text-sm text-slate-500 leading-relaxed max-w-md">
                {isRejected
                  ? "Please review the rejection reason provided below, update your details, and resubmit."
                  : isApproved
                  ? "You can now create and manage events directly from your dashboard."
                  : "Our team is reviewing your business information and verification documents. You will be notified once a decision has been made."}
              </p>

              {/* Prominent Rejection Reason Box on Status Page */}
              {isRejected && (
                <div className="mt-6 w-full bg-red-50/90 border border-red-200 rounded-2xl p-4 sm:p-5 text-left shadow-xs">
                  <span className="text-[11px] font-bold uppercase tracking-wider text-red-800 block mb-1">
                    Rejection Reason
                  </span>
                  <p className="text-sm font-semibold text-red-950 leading-relaxed">
                    {rejectionReason || "Please correct your business verification document."}
                  </p>
                  <p className="text-xs text-red-700 mt-2">
                    You can update your application details and resubmit it.
                  </p>
                </div>
              )}
            </div>

            {/* Status Timeline - 3 stages (Submitted -> Under Review -> Approved) */}
            <div className="pt-8 border-t border-slate-100">
              <div className="relative flex items-center justify-between max-w-lg mx-auto px-4">
                {/* Background line */}
                <div className="absolute top-4 left-8 right-8 h-1 bg-slate-200 -z-0 rounded-full" />
                {/* Active line progress */}
                <div
                  className={`absolute top-4 left-8 h-1 rounded-full transition-all -z-0 ${
                    isApproved ? "w-[calc(100%-4rem)] bg-emerald-500" : isRejected ? "w-1/2 bg-red-400" : "w-1/2 bg-primary"
                  }`}
                />

                {/* Stage 1: Submitted (Completed) */}
                <div className="flex flex-col items-center gap-2 z-10">
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center text-white shadow-sm ${
                      isApproved ? "bg-emerald-500" : "bg-primary"
                    }`}
                  >
                    <svg width="12" height="10" viewBox="0 0 12 10" fill="none" aria-hidden="true">
                      <path d="M1.5 5L4.5 8L10.5 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  </div>
                  <span className="text-xs font-semibold text-slate-900 tracking-tight text-center">
                    Submitted
                  </span>
                </div>

                {/* Stage 2: Under Review / Rejected */}
                <div className="flex flex-col items-center gap-2 z-10">
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center shadow-sm ${
                      isApproved
                        ? "bg-emerald-500 text-white"
                        : isRejected
                        ? "bg-red-500 text-white"
                        : "bg-white border-2 border-primary"
                    }`}
                  >
                    {isApproved ? (
                      <svg width="12" height="10" viewBox="0 0 12 10" fill="none" aria-hidden="true">
                        <path d="M1.5 5L4.5 8L10.5 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                    ) : isRejected ? (
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                        <line x1="18" y1="6" x2="6" y2="18" />
                        <line x1="6" y1="6" x2="18" y2="18" />
                      </svg>
                    ) : (
                      <div className="w-2.5 h-2.5 rounded-full bg-primary animate-ping" />
                    )}
                  </div>
                  <span
                    className={`text-xs font-bold tracking-tight text-center ${
                      isApproved
                        ? "text-slate-900"
                        : isRejected
                        ? "text-red-600"
                        : "text-primary"
                    }`}
                  >
                    {isRejected ? "Rejected" : "Under Review"}
                  </span>
                </div>

                {/* Stage 3: Approved */}
                <div className={`flex flex-col items-center gap-2 z-10 ${isApproved ? "" : "opacity-50"}`}>
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center shadow-sm ${
                      isApproved
                        ? "bg-emerald-500 text-white"
                        : "bg-white border-2 border-slate-300"
                    }`}
                  >
                    {isApproved ? (
                      <svg width="12" height="10" viewBox="0 0 12 10" fill="none" aria-hidden="true">
                        <path d="M1.5 5L4.5 8L10.5 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                    ) : (
                      <div className="w-2 h-2 rounded-full bg-slate-300" />
                    )}
                  </div>
                  <span
                    className={`text-xs tracking-tight text-center ${
                      isApproved ? "font-bold text-emerald-700" : "font-medium text-slate-500"
                    }`}
                  >
                    Approved
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Bottom Section: Summary Card & Actions */}
        <div className="grid grid-cols-1 md:grid-cols-12 gap-5 items-stretch">
          {/* Application Summary Card */}
          <div className="md:col-span-8 bg-white rounded-2xl border border-slate-200/80 shadow-sm p-5 sm:p-6 flex flex-col justify-between">
            <div>
              <div className="flex items-center gap-2 pb-3 mb-4 border-b border-slate-100 text-slate-800">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-primary" aria-hidden="true">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                  <line x1="16" y1="13" x2="8" y2="13" />
                  <line x1="16" y1="17" x2="8" y2="17" />
                  <polyline points="10 9 9 9 8 9" />
                </svg>
                <h3 className="font-display text-base font-bold text-slate-900">
                  Application Summary
                </h3>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <span className="text-xs font-medium text-slate-400">Business Name</span>
                  <p className="text-sm font-semibold text-slate-800 truncate">{businessName}</p>
                </div>
                <div>
                  <span className="text-xs font-medium text-slate-400">Business Type</span>
                  <p className="text-sm font-semibold text-slate-800">{businessType}</p>
                </div>
                <div>
                  <span className="text-xs font-medium text-slate-400">Application ID</span>
                  <p className="text-sm font-semibold text-slate-800 font-mono break-all">{applicationId || "—"}</p>
                </div>
                <div>
                  <span className="text-xs font-medium text-slate-400">Date Submitted</span>
                  <p className="text-sm font-semibold text-slate-800">{formattedDate}</p>
                </div>
              </div>
            </div>

            {isRejected && rejectionReason && (
              <div className="mt-4 pt-3 border-t border-slate-100">
                <span className="text-xs font-bold text-red-600 uppercase tracking-wide">
                  Rejection Note
                </span>
                <p className="text-xs sm:text-sm text-slate-700 mt-0.5 font-medium">
                  {rejectionReason}
                </p>
              </div>
            )}
          </div>

          {/* Actions Column */}
          <div className="md:col-span-4 flex flex-col justify-center gap-3">
            {isRejected ? (
              <button
                type="button"
                onClick={() => navigate("/become-organizer")}
                className="w-full h-11 px-5 rounded-lg bg-primary hover:bg-primary/90 text-white text-sm font-semibold transition-colors flex items-center justify-center gap-2 shadow-sm cursor-pointer"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                </svg>
                Edit & Resubmit Application
              </button>
            ) : isApproved ? (
              <button
                type="button"
                onClick={() => navigate("/profile")}
                className="w-full h-11 px-5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-semibold transition-colors flex items-center justify-center gap-2 shadow-sm cursor-pointer"
              >
                Go to Profile
              </button>
            ) : null}

            <button
              type="button"
              onClick={() => setIsModalOpen(true)}
              className={`w-full h-11 px-5 rounded-lg text-sm font-semibold transition-colors flex items-center justify-center gap-2 shadow-sm cursor-pointer ${
                isRejected
                  ? "border border-slate-300 bg-white hover:bg-slate-50 text-slate-700"
                  : "bg-primary hover:bg-primary/90 text-white"
              }`}
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
              View Application
            </button>

            <button
              type="button"
              onClick={() => navigate("/profile")}
              className="w-full h-11 px-5 rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-sm font-semibold transition-colors flex items-center justify-center gap-2 shadow-sm cursor-pointer"
            >
              <svg width="14" height="14" viewBox="0 0 12 12" fill="none" aria-hidden="true">
                <path d="M9.5 6H2.5M5.5 3L2.5 6L5.5 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
              Back to Profile
            </button>
          </div>
        </div>
      </main>

      <Footer />

      {/* View Application Details Modal */}
      <ApplicationDetailsModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </div>
  );
}
