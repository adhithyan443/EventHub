export default function PublishEventConfirmationModal({
  isOpen,
  onClose,
  onConfirm,
  eventTitle,
  loading = false,
  error = "",
}) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-xs animate-fade-in"
      onClick={() => {
        if (!loading) onClose();
      }}
    >
      <div
        className="bg-white rounded-2xl shadow-2xl border border-[#bcc9c6] max-w-md w-full overflow-hidden"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="publish-event-title"
      >
        <div className="p-6">
          <div className="flex items-start gap-4">
            <div className="size-11 rounded-full bg-[#00685f]/10 border border-[#00685f]/20 flex items-center justify-center shrink-0 text-[#00685f]">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="size-6 text-[#00685f]"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="2" y1="12" x2="22" y2="12" />
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
              </svg>
            </div>

            <div className="flex-1">
              <h3
                id="publish-event-title"
                className="text-lg font-bold text-[#141b2b]"
              >
                Make Event Public
              </h3>

              <p className="mt-2 text-sm text-[#565e74] leading-relaxed">
                Are you sure you want to publish{" "}
                <span className="font-semibold text-[#141b2b]">
                  {eventTitle ? `"${eventTitle}"` : "this event"}
                </span>
                ?
              </p>

              <div className="mt-3 p-3 bg-[#00685f]/5 border border-[#00685f]/20 rounded-lg text-xs text-[#00685f] leading-relaxed">
                <span className="font-semibold">Notice:</span> Publishing will make this event publicly visible to attendees on EventHub. Attendees will be able to discover the event and purchase tickets.
              </div>

              {error && (
                <div className="mt-3 p-3 bg-red-100 border border-red-200 rounded-lg text-xs text-red-800 font-medium">
                  {error}
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="bg-[#f1f3ff]/60 px-6 py-4 border-t border-[#bcc9c6]/40 flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="px-4 py-2 text-xs sm:text-sm font-medium text-[#565e74] hover:text-[#141b2b] bg-white border border-[#bcc9c6] hover:bg-gray-50 rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>

          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            className="px-4 py-2 text-xs sm:text-sm font-medium text-white bg-[#00685f] hover:bg-[#005a52] rounded-lg transition-colors shadow-2xs cursor-pointer flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <>
                <svg
                  className="animate-spin size-4 text-white"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                <span>Publishing...</span>
              </>
            ) : (
              <span>Make Event Public</span>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
