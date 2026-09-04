import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import useOrganizerStore from "../../store/organizerStore";
import ApplicationDetailsModal from "./ApplicationDetailsModal";

export default function OrganizerStatusPage() {
  const navigate = useNavigate();
  const [isModalOpen, setIsModalOpen] = useState(false);

  const businessName = useOrganizerStore((state) => state.businessName) || "Event Masters Pvt Ltd";
  const businessType = useOrganizerStore((state) => state.businessType) || "Company";
  const applicationId = useOrganizerStore((state) => state.applicationId) || "APP001";
  const applicationStatus = useOrganizerStore((state) => state.applicationStatus) || "PENDING";
  const submissionDate = useOrganizerStore((state) => state.submissionDate) || "10 August 2026";

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
                <p className="font-display text-xl sm:text-2xl font-bold text-slate-900">
                  {applicationId}
                </p>
              </div>

              <div className="flex sm:flex-col sm:items-end justify-between items-center gap-1">
                <span className="text-xs text-slate-500 font-medium">
                  {submissionDate}
                </span>
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold tracking-wide bg-primary/10 text-primary border border-primary/20">
                  <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
                  {applicationStatus}
                </span>
              </div>
            </div>

            {/* Hero Message */}
            <div className="py-8 sm:py-10 flex flex-col items-center text-center max-w-lg mx-auto">
              <div className="w-20 h-20 sm:w-24 sm:h-24 rounded-full bg-primary/10 flex items-center justify-center text-primary mb-6 shadow-xs">
                <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                  <circle cx="9" cy="7" r="4" />
                  <polyline points="16 11 18 13 22 9" />
                </svg>
              </div>

              <h2 className="font-display text-xl sm:text-2xl font-bold text-slate-900 mb-2">
                Application Under Review
              </h2>
              <p className="text-sm sm:text-base font-semibold text-slate-800 mb-2">
                Your organizer application has been submitted and is currently being reviewed.
              </p>
              <p className="text-xs sm:text-sm text-slate-500 leading-relaxed max-w-md">
                Our team is reviewing your business information and verification documents. You will be notified once a decision has been made.
              </p>
            </div>

            {/* Status Timeline - EXACTLY 3 stages (Submitted -> Under Review -> Approved) */}
            <div className="pt-8 border-t border-slate-100">
              <div className="relative flex items-center justify-between max-w-lg mx-auto px-4">
                {/* Background line */}
                <div className="absolute top-4 left-8 right-8 h-1 bg-slate-200 -z-0 rounded-full" />
                {/* Active line from Step 1 to Step 2 (50% progress) */}
                <div className="absolute top-4 left-8 w-1/2 h-1 bg-primary -z-0 rounded-full transition-all" />

                {/* Stage 1: Submitted (Completed) */}
                <div className="flex flex-col items-center gap-2 z-10">
                  <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-white shadow-sm">
                    <svg width="12" height="10" viewBox="0 0 12 10" fill="none" aria-hidden="true">
                      <path d="M1.5 5L4.5 8L10.5 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  </div>
                  <span className="text-xs font-semibold text-slate-900 tracking-tight text-center">
                    Submitted
                  </span>
                </div>

                {/* Stage 2: Under Review (Active) */}
                <div className="flex flex-col items-center gap-2 z-10">
                  <div className="w-8 h-8 rounded-full bg-white border-2 border-primary flex items-center justify-center shadow-sm">
                    <div className="w-2.5 h-2.5 rounded-full bg-primary animate-ping" />
                  </div>
                  <span className="text-xs font-bold text-primary tracking-tight text-center">
                    Under Review
                  </span>
                </div>

                {/* Stage 3: Approved (Pending) */}
                <div className="flex flex-col items-center gap-2 z-10 opacity-50">
                  <div className="w-8 h-8 rounded-full bg-white border-2 border-slate-300 flex items-center justify-center">
                    <div className="w-2 h-2 rounded-full bg-slate-300" />
                  </div>
                  <span className="text-xs font-medium text-slate-500 tracking-tight text-center">
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
                <p className="text-sm font-semibold text-slate-800 font-mono">{applicationId}</p>
              </div>
              <div>
                <span className="text-xs font-medium text-slate-400">Date Submitted</span>
                <p className="text-sm font-semibold text-slate-800">{submissionDate}</p>
              </div>
            </div>
          </div>

          {/* Actions Column */}
          <div className="md:col-span-4 flex flex-col justify-center gap-3">
            <button
              type="button"
              onClick={() => setIsModalOpen(true)}
              className="w-full h-11 px-5 rounded-lg bg-primary hover:bg-primary/90 text-white text-sm font-semibold transition-colors flex items-center justify-center gap-2 shadow-sm cursor-pointer"
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
