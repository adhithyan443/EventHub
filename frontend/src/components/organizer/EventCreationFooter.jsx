import { useState } from "react";

export default function EventCreationFooter({
  onBack,
  onContinue,
  isFirstStep = false,
  isReviewStep = false,
  continueLabel,
}) {
  const [draftSavedToast, setDraftSavedToast] = useState(false);

  const handleSaveDraft = () => {
    setDraftSavedToast(true);
    setTimeout(() => {
      setDraftSavedToast(false);
    }, 2500);
  };

  const primaryLabel =
    continueLabel || (isReviewStep ? "🚀 Publish Event" : "Save & Continue");

  return (
    <footer className="sticky bottom-0 left-0 right-0 z-20 bg-white/95 backdrop-blur border-t border-[#bcc9c6]/50 px-8 py-4 shadow-[0_-4px_12px_rgba(0,0,0,0.03)] flex items-center justify-between">
      {/* Toast Notification for Draft */}
      {draftSavedToast && (
        <div className="absolute top-[-44px] left-1/2 -translate-x-1/2 bg-[#00685f] text-white text-xs font-semibold px-4 py-2 rounded-full shadow-lg flex items-center gap-2 animate-bounce">
          <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M5 13l4 4L19 7" />
          </svg>
          <span>Progress saved as draft locally</span>
        </div>
      )}

      {/* Left side: Back button */}
      <div>
        {!isFirstStep && onBack ? (
          <button
            type="button"
            onClick={onBack}
            className="flex items-center gap-2 border border-[#bcc9c6] bg-white hover:bg-gray-100/80 text-[#141b2b] text-[14px] font-semibold px-5 py-2.5 rounded-lg transition-colors cursor-pointer"
          >
            <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            <span>Back</span>
          </button>
        ) : (
          <div />
        )}
      </div>

      {/* Right side: Save as Draft & Primary Continue */}
      <div className="flex items-center gap-4">
        {!isReviewStep && (
          <button
            type="button"
            onClick={handleSaveDraft}
            className="text-[14px] font-semibold text-[#00685f] hover:text-[#005550] px-3 py-2 transition-colors cursor-pointer"
          >
            Save as Draft
          </button>
        )}

        <button
          type="button"
          onClick={onContinue}
          className="flex items-center gap-2 bg-[#00685f] hover:bg-[#005550] text-white text-[14px] font-semibold px-6 py-2.5 rounded-lg shadow-sm transition-all cursor-pointer"
        >
          <span>{primaryLabel}</span>
          {!isReviewStep && (
            <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
            </svg>
          )}
        </button>
      </div>
    </footer>
  );
}
