export default function UnblockUserModal({ user, onConfirm, onCancel, loading = false }) {
  if (!user) return null;

  const displayName = user.full_name || user.name || "User";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={loading ? undefined : onCancel}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[480px] overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <div className="p-4 sm:p-6">
          {/* Header */}
          <div className="flex items-start gap-4 mb-5">
            <div className="bg-[#dcfce7] w-10 h-10 rounded-full flex items-center justify-center shrink-0 mt-0.5">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <circle cx="10" cy="10" r="9" stroke="#166534" strokeWidth="1.5" />
                <path
                  d="M6 10L8.5 12.5L14 7"
                  stroke="#166534"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </div>
            <div>
              <h2 className="text-[#141b2b] text-xl font-bold leading-7">
                Unblock User Account?
              </h2>
              <p className="text-[#3d4947] text-sm mt-1 leading-5">
                You are about to reactivate{" "}
                <span className="font-semibold text-[#141b2b]">{displayName}</span>{" "}
                (<span className="font-mono text-xs">{user.email}</span>).
                They will regain normal platform access and the ability to log in and make bookings.
              </p>
            </div>
          </div>

          {/* Actions */}
          <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-2 sm:gap-3 mt-6">
            <button
              type="button"
              onClick={onCancel}
              disabled={loading}
              className="w-full sm:w-auto border border-[#bcc9c6] bg-white text-[#141b2b] text-sm font-medium px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#f9f9ff] transition-colors cursor-pointer disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={loading}
              onClick={() => onConfirm(user.id)}
              className="w-full sm:w-auto bg-[#00685f] text-white text-sm font-semibold px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#00524b] transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2 cursor-pointer shadow-sm"
            >
              {loading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  <span>Reactivating...</span>
                </>
              ) : (
                <>
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path
                      d="M11.6666 3.5L5.24992 9.91667L2.33325 7"
                      stroke="white"
                      strokeWidth="1.75"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                  Confirm Unblock
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
