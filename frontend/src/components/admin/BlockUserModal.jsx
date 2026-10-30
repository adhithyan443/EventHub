import { useState } from "react";

export default function BlockUserModal({ user, onConfirm, onCancel, loading = false }) {
  const [reason, setReason] = useState("");

  if (!user) return null;

  const handleSubmit = (e) => {
    e.preventDefault();
    if (reason.trim()) {
      onConfirm(user.id, reason.trim());
    }
  };

  const displayName = user.full_name || user.name || "User";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={loading ? undefined : onCancel}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[500px] overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <form onSubmit={handleSubmit} className="p-4 sm:p-6">
          {/* Header */}
          <div className="flex items-start gap-4 mb-5">
            <div className="bg-[#ffdad6] w-10 h-10 rounded-full flex items-center justify-center shrink-0 mt-0.5">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path
                  d="M10 1.5L1 18H19L10 1.5Z"
                  stroke="#93000a"
                  strokeWidth="1.5"
                  fill="none"
                />
                <path
                  d="M10 8V12"
                  stroke="#93000a"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                />
                <circle cx="10" cy="15" r="0.75" fill="#93000a" />
              </svg>
            </div>
            <div>
              <h2 className="text-[#141b2b] text-xl font-bold leading-7">
                Block User Account?
              </h2>
              <p className="text-[#3d4947] text-sm mt-1 leading-5">
                You are about to block{" "}
                <span className="font-semibold text-[#141b2b]">{displayName}</span>{" "}
                (<span className="font-mono text-xs">{user.email}</span>).
                All active sessions will be terminated immediately, and they will be prevented from accessing protected services.
              </p>
            </div>
          </div>

          {/* Reason Input */}
          <div className="mb-5">
            <label className="block text-sm font-semibold text-[#141b2b] mb-2">
              Reason for suspension <span className="text-[#93000a]">(Required)</span>
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Provide a clear reason for blocking this account..."
              rows={3}
              required
              disabled={loading}
              className="w-full border border-[#bcc9c6] rounded-lg px-3 py-2.5 text-sm text-[#141b2b] placeholder:text-[#6d7a77] resize-none focus:outline-none focus:ring-2 focus:ring-[#00685f]/30 focus:border-[#00685f] transition-colors disabled:opacity-50"
            />
          </div>

          {/* Actions */}
          <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-2 sm:gap-3">
            <button
              type="button"
              onClick={onCancel}
              disabled={loading}
              className="w-full sm:w-auto border border-[#bcc9c6] bg-white text-[#141b2b] text-sm font-medium px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#f9f9ff] transition-colors cursor-pointer disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!reason.trim() || loading}
              className="w-full sm:w-auto bg-[#93000a] text-white text-sm font-semibold px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#7d000a] transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2 cursor-pointer shadow-sm"
            >
              {loading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  <span>Blocking...</span>
                </>
              ) : (
                <>
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path
                      d="M7 0C3.13 0 0 3.13 0 7C0 10.87 3.13 14 7 14C10.87 14 14 10.87 14 7C14 3.13 10.87 0 7 0ZM8.5 10.5H5.5V6.5H8.5V10.5ZM8.5 5H5.5V2H8.5V5Z"
                      fill="white"
                    />
                  </svg>
                  Confirm Block
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
