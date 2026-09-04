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


export default function BecomeOrganizerPage() {
  const navigate = useNavigate();

  const currentStep = useOrganizerStore((state) => state.currentStep);
  const setCurrentStep = useOrganizerStore((state) => state.setCurrentStep);
  const saveDraft = useOrganizerStore((state) => state.saveDraft);


  const applicationSubmitted = useOrganizerStore(
    (state) => state.applicationSubmitted
  );

  const submitApplication = useOrganizerStore(
    (state) => state.submitApplication
  );


  const [toastMessage, setToastMessage] = useState("");

  // If application already submitted, redirect to status page
  useEffect(() => {
    if (applicationSubmitted) {
      navigate("/become-organizer/status", { replace: true });
    }
  }, [applicationSubmitted, navigate]);

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
