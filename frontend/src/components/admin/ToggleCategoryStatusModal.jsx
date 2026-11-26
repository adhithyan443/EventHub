export default function ToggleCategoryStatusModal({
  category,
  targetStatus,
  isOpen,
  onClose,
  onConfirm,
  loading = false,
}) {
  if (!isOpen || !category) return null;

  const isDeactivating = targetStatus === "INACTIVE";
  const eventCount = category.event_count || 0;

  const handleConfirm = () => {
    onConfirm(category.id, targetStatus);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={loading ? undefined : onClose}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[480px] overflow-hidden animate-in fade-in zoom-in-95 duration-150 border border-[#dce2f7]">
        <div className="p-4 sm:p-6">
          {/* Header Icon & Title */}
          <div className="flex items-start gap-4 mb-4">
            <div
              className={`w-11 h-11 rounded-full flex items-center justify-center shrink-0 mt-0.5 ${
                isDeactivating
                  ? "bg-[#ffdad6] text-[#ba1a1a]"
                  : "bg-[#e6f4f2] text-[#00685f]"
              }`}
            >
              {isDeactivating ? (
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                  />
                </svg>
              ) : (
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              )}
            </div>
            <div>
              <h2 className="text-[#141b2b] text-lg font-bold leading-6">
                {isDeactivating ? "Deactivate Category?" : "Activate Category?"}
              </h2>
              <p className="text-[#3d4947] text-xs mt-1">
                Category: <span className="font-semibold text-[#141b2b]">{category.name}</span>
              </p>
            </div>
          </div>

          {/* Impact & Context Explanation */}
          <div className="space-y-3 mb-6 text-sm text-[#3d4947] leading-relaxed">
            {isDeactivating ? (
              <>
                <p>
                  You are about to deactivate the category{" "}
                  <strong className="text-[#141b2b]">"{category.name}"</strong>.
                </p>

                {/* Event Relationship Indicator */}
                <div className="rounded-xl border border-[#dce2f7] bg-[#f9f9ff] p-3.5 space-y-1.5">
                  <div className="flex items-center justify-between text-xs font-semibold text-[#141b2b]">
                    <span>Associated Events</span>
                    <span className="px-2 py-0.5 rounded-md bg-[#eef0fb] text-[#141b2b] font-bold">
                      {eventCount} {eventCount === 1 ? "Event" : "Events"}
                    </span>
                  </div>
                  <p className="text-xs text-[#6d7a77]">
                    {eventCount > 0
                      ? "Existing published events will remain fully accessible, viewable, and discoverable. Their historical data is never deleted or altered."
                      : "No events are currently associated with this category."}
                  </p>
                </div>

                <div className="rounded-lg bg-[#fff5f4] border border-[#ffdad6] p-3 text-xs text-[#93000a] flex items-start gap-2">
                  <svg className="w-4 h-4 shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                    <path
                      fillRule="evenodd"
                      d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span>
                    Organizers will no longer be able to select this category when creating new events or publishing draft events.
                  </span>
                </div>
              </>
            ) : (
              <div className="rounded-xl border border-[#dce2f7] bg-[#f9f9ff] p-3.5 text-xs text-[#3d4947] leading-relaxed">
                Activating this category will immediately make it visible in the public categories list and available for organizers to select during event creation.
              </div>
            )}
          </div>

          {/* Action Buttons */}
          <div className="flex items-center justify-end gap-3 pt-3 border-t border-[#dce2f7]">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="px-4 py-2.5 text-sm font-semibold text-[#141b2b] hover:bg-[#eef0fb] rounded-xl transition-colors cursor-pointer disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleConfirm}
              disabled={loading}
              className={`px-5 py-2.5 text-sm font-semibold text-white rounded-xl transition-colors shadow-sm disabled:opacity-50 cursor-pointer flex items-center gap-2 ${
                isDeactivating
                  ? "bg-[#ba1a1a] hover:bg-[#93000a]"
                  : "bg-[#00685f] hover:bg-[#00574f]"
              }`}
            >
              {loading ? (
                <>
                  <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                  <span>Processing...</span>
                </>
              ) : (
                <span>{isDeactivating ? "Confirm Deactivation" : "Confirm Activation"}</span>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
