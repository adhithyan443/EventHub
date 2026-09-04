import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import Stepper from "./Stepper";
import Step1BusinessInfo from "./Step1BusinessInfo";
import Step2Verification from "./Step2Verification";
import Step3BankDetails from "./Step3BankDetails";
import Step4ReviewSubmit from "./Step4ReviewSubmit";
import useOrganizerStore from "../../store/organizerStore";
import useAuthStore from "../../store/authStore";

export default function BecomeOrganizerPage() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);

  const currentStep = useOrganizerStore((state) => state.currentStep);
  const setCurrentStep = useOrganizerStore((state) => state.setCurrentStep);
  const saveDraft = useOrganizerStore((state) => state.saveDraft);

  const applicationSubmitted = useOrganizerStore(
    (state) => state.applicationSubmitted
  );
  const applicationStatus = useOrganizerStore(
    (state) => state.applicationStatus
  );
  const rejectionReason = useOrganizerStore(
    (state) => state.rejectionReason
  );
  const isCheckingApplication = useOrganizerStore(
    (state) => state.isCheckingApplication
  );
  const fetchError = useOrganizerStore(
    (state) => state.fetchError
  );
  const fetchApplication = useOrganizerStore(
    (state) => state.fetchApplication
  );
  const submitApplication = useOrganizerStore(
    (state) => state.submitApplication
  );

  const [toastMessage, setToastMessage] = useState("");

  // Check application status from backend on mount
  useEffect(() => {
    // If user is already an organizer, redirect to profile/organizer area
    if (user?.role === "ORGANIZER") {
      navigate("/profile", { replace: true });
      return;
    }

    fetchApplication().then((result) => {
      if (result?.exists && result?.application) {
        const status = result.application.status;
        if (status === "PENDING") {
          navigate("/become-organizer/status", { replace: true });
        } else if (status === "APPROVED") {
          navigate("/profile", { replace: true });
        }
        // If REJECTED, user stays on /become-organizer with loaded application!
      }
    });
  }, [user, navigate, fetchApplication]);

  // Scroll to top on step changes
  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "smooth" });
  }, [currentStep]);

  function showToast(msg) {
    setToastMessage(msg);

    setTimeout(() => {
      setToastMessage("");
    }, 4500);
  }

  function handleSaveLater() {
    const success = saveDraft();
    if (success) {
      showToast(
        "Draft saved successfully! Non-sensitive information has been preserved. Sensitive details remain protected."
      );
    } else {
      showToast("Unable to save draft locally. Please ensure storage permissions are enabled.");
    }
  }

  function handleNextStep() {
    setCurrentStep(Math.min(currentStep + 1, 4));
  }

  function handlePrevStep() {
    setCurrentStep(Math.max(currentStep - 1, 1));
  }

  function handleEditStep(stepNum) {
    setCurrentStep(stepNum);
  }

  async function handleSubmit() {
    try {
      await submitApplication();
      navigate("/become-organizer/status");
    } catch {
      // Step 4 will display submitError from the store and user stays on Step 4
    }
  }

  // Loading state while checking backend application status
  if (isCheckingApplication) {
    return (
      <div className="min-h-screen bg-[#F9F9FF] flex flex-col justify-between">
        <AppHeader />
        <main className="flex-1 flex items-center justify-center p-6">
          <div className="flex flex-col items-center gap-3 text-center">
            <div className="w-9 h-9 border-3 border-primary border-t-transparent rounded-full animate-spin" />
            <p className="text-sm font-semibold text-slate-700">Checking your application status...</p>
            <p className="text-xs text-slate-400">Verifying with EventHub servers</p>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  // Error state if checking application status failed with 500 or network error
  if (fetchError && !applicationStatus) {
    return (
      <div className="min-h-screen bg-[#F9F9FF] flex flex-col justify-between">
        <AppHeader />
        <main className="flex-1 flex items-center justify-center p-6">
          <div className="max-w-md w-full bg-white rounded-2xl border border-slate-200 p-6 text-center shadow-sm">
            <div className="w-12 h-12 rounded-full bg-red-50 text-red-500 mx-auto flex items-center justify-center mb-4">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </div>
            <h2 className="font-display text-lg font-bold text-slate-900 mb-1">
              Unable to Check Application Status
            </h2>
            <p className="text-xs text-slate-600 mb-6 leading-relaxed">
              {fetchError}. Please verify your connection or try again.
            </p>
            <div className="flex flex-col sm:flex-row gap-3">
              <button
                type="button"
                onClick={() => fetchApplication()}
                className="flex-1 h-10 rounded-lg bg-primary hover:bg-primary/90 text-white text-sm font-semibold transition-colors cursor-pointer shadow-sm"
              >
                Retry
              </button>
              <button
                type="button"
                onClick={() => navigate("/profile")}
                className="flex-1 h-10 rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-sm font-semibold transition-colors cursor-pointer shadow-sm"
              >
                Back to Profile
              </button>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#F9F9FF] flex flex-col justify-between">
      <AppHeader />

      {/* Floating Toast Notification */}
      {toastMessage && (
        <div className="fixed top-20 right-4 sm:right-8 z-50 max-w-md bg-slate-900 text-white px-5 py-3.5 rounded-xl shadow-xl flex items-start gap-3 border border-slate-700 animate-slide-in">
          <div className="text-emerald-400 shrink-0 mt-0.5">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M20 6L9 17l-5-5" />
            </svg>
          </div>
          <div className="flex-1 text-xs sm:text-sm leading-snug">
            {toastMessage}
          </div>
          <button
            type="button"
            onClick={() => setToastMessage("")}
            className="text-slate-400 hover:text-white shrink-0 -mr-1 -mt-1 p-1"
            aria-label="Dismiss notification"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      )}

      <main className="flex-1 max-w-3xl mx-auto w-full px-4 sm:px-6 py-8 sm:py-10">
        {/* Prominent Rejection Banner for Rejected Applications */}
        {applicationStatus === "REJECTED" && (
          <div className="mb-8 rounded-2xl bg-gradient-to-r from-amber-50 via-orange-50/50 to-amber-50 border border-amber-200/90 p-5 sm:p-6 shadow-xs animate-fade-in">
            <div className="flex items-start gap-3.5">
              <div className="w-10 h-10 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-600 flex items-center justify-center shrink-0 mt-0.5">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="12" />
                  <line x1="12" y1="16" x2="12.01" y2="16" />
                </svg>
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <h2 className="font-display text-base sm:text-lg font-bold text-amber-950">
                    Application Rejected
                  </h2>
                  <span className="px-2.5 py-0.5 rounded-full text-[11px] font-bold tracking-wide uppercase bg-amber-200/80 text-amber-900 border border-amber-300">
                    Action Required
                  </span>
                </div>

                <div className="mt-2 text-xs sm:text-sm text-amber-900 leading-relaxed bg-white/80 rounded-xl p-3.5 border border-amber-200/60">
                  <span className="font-bold text-amber-950 uppercase text-[11px] tracking-wider block mb-1">
                    Rejection Reason
                  </span>
                  <p className="font-medium text-amber-900">
                    {rejectionReason || "Please review your submitted documents and details for accuracy."}
                  </p>
                </div>

                <p className="mt-3 text-xs sm:text-sm text-amber-800 leading-relaxed">
                  Your previously submitted information has been restored across all 4 steps below. Please update the necessary details and resubmit your application.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Progress Stepper */}
        <div className="mb-8 sm:mb-10">
          <Stepper currentStep={currentStep} />
        </div>

        {/* Step Views */}
        {currentStep === 1 && (
          <Step1BusinessInfo
            onNext={handleNextStep}
            onSaveLater={handleSaveLater}
          />
        )}

        {currentStep === 2 && (
          <Step2Verification
            onNext={handleNextStep}
            onBack={handlePrevStep}
            onSaveLater={handleSaveLater}
          />
        )}

        {currentStep === 3 && (
          <Step3BankDetails
            onNext={handleNextStep}
            onBack={handlePrevStep}
            onSaveLater={handleSaveLater}
          />
        )}

        {currentStep === 4 && (
          <Step4ReviewSubmit
            onSubmit={handleSubmit}
            onBack={handlePrevStep}
            onSaveLater={handleSaveLater}
            onEditStep={handleEditStep}
          />
        )}
      </main>

      <Footer />
    </div>
  );
}
