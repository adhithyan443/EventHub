import notify from "../../utils/toast";

export default function EventCreationFooter({
  onBack,
  onContinue,
  onSaveDraft,
  isFirstStep = false,
  isReviewStep = false,
  continueLabel,
  saveDraftLabel = "Save as Draft",
  isPublishing = false,
  isSubmittingDraft = false,
  disabled = false,
}) {
  const handleSaveDraft = () => {
    if (onSaveDraft) {
      onSaveDraft();
    } else {
      notify.success("Progress saved as draft locally.");
    }
  };

  const isBusy = isPublishing || isSubmittingDraft;

  const primaryLabel =
    continueLabel || (isReviewStep ? "🚀 Publish Event" : "Save & Continue");

  return (
    <footer className="sticky bottom-0 left-0 right-0 z-20 bg-white/95 backdrop-blur border-t border-[#bcc9c6]/50 px-4 sm:px-8 py-3 sm:py-4 shadow-[0_-4px_12px_rgba(0,0,0,0.03)] flex items-center justify-between gap-2 sm:gap-4">
      {/* Left side: Back button */}
      <div>
        {!isFirstStep && onBack ? (
          <button
            type="button"
            onClick={onBack}
            disabled={disabled || isBusy}
            className="flex items-center gap-1.5 sm:gap-2 border border-[#bcc9c6] bg-white hover:bg-gray-100/80 text-[#141b2b] text-xs sm:text-[14px] font-semibold px-3 sm:px-5 py-2 sm:py-2.5 rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg className="size-3.5 sm:size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            <span>Back</span>
          </button>
        ) : (
          <div />
        )}
      </div>

      {/* Right side: Save as Draft & Primary Continue */}
      <div className="flex items-center gap-2 sm:gap-4">
        {/* On review step: Explicit Save as Draft button. On earlier steps: Quick draft action */}
        {isReviewStep ? (
          <button
            type="button"
            onClick={handleSaveDraft}
            disabled={disabled || isBusy}
            className="flex items-center gap-1.5 sm:gap-2 border border-[#00685f] bg-white hover:bg-[#00685f]/5 text-[#00685f] text-xs sm:text-[14px] font-semibold px-4 sm:px-5 py-2 sm:py-2.5 rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmittingDraft ? (
              <>
                <span className="size-3.5 border-2 border-[#00685f] border-t-transparent rounded-full animate-spin" />
                <span>Saving Draft...</span>
              </>
            ) : (
              <span>{saveDraftLabel}</span>
            )}
          </button>
        ) : (
          <button
            type="button"
            onClick={handleSaveDraft}
            disabled={disabled || isBusy}
            className="text-xs sm:text-[14px] font-semibold text-[#00685f] hover:text-[#005550] px-2 sm:px-3 py-1.5 sm:py-2 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <span className="hidden sm:inline">Save as Draft</span>
            <span className="sm:hidden">Draft</span>
          </button>
        )}

        <button
          type="button"
          onClick={onContinue}
          disabled={disabled || isBusy}
          className="flex items-center gap-1.5 sm:gap-2 bg-[#00685f] hover:bg-[#005550] text-white text-xs sm:text-[14px] font-semibold px-4 sm:px-6 py-2 sm:py-2.5 rounded-lg shadow-sm transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isPublishing ? (
            <>
              <span className="size-3.5 border-2 border-white border-t-transparent rounded-full animate-spin" />
              <span>Publishing Event...</span>
            </>
          ) : (
            <>
              <span>{primaryLabel}</span>
              {!isReviewStep && (
                <svg className="size-3.5 sm:size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                </svg>
              )}
            </>
          )}
        </button>
      </div>
    </footer>
  );
}
