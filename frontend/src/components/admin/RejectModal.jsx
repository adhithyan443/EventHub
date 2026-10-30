import { useState } from "react";

export default function RejectModal({
  appId,
  businessName,
  actionLoading = false,
  onConfirm,
  onCancel,
}) {
  const [reason, setReason] = useState("");

  const handleSubmit = (e) => {
    e.preventDefault();
    if (reason.trim() && !actionLoading) {
      onConfirm(reason.trim());
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={!actionLoading ? onCancel : undefined}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[500px] overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <form onSubmit={handleSubmit} className="p-4 sm:p-6">
          {/* Header */}
          <div className="flex items-start gap-4 mb-5">
            <div className="bg-[#fef3c7] w-10 h-10 rounded-full flex items-center justify-center shrink-0 mt-0.5">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path
                  d="M10 1.5L1 18H19L10 1.5Z"
                  stroke="#92400e"
                  strokeWidth="1.5"
                  fill="none"
                />
                <path
                  d="M10 8V12"
                  stroke="#92400e"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                />
                <circle cx="10" cy="15" r="0.75" fill="#92400e" />
              </svg>
            </div>
            <div>
              <h2 className="text-[#141b2b] text-xl font-bold leading-7">
                Reject Application?
              </h2>
              <p className="text-[#3d4947] text-sm mt-1 leading-5">
                You are about to reject the application from{" "}
                <span className="font-semibold text-[#141b2b]">{businessName}</span>{" "}
                (<span className="font-mono text-xs">{appId}</span>). The applicant will
                be notified with your reason.
              </p>
            </div>
          </div>

          {/* Reason Input */}
          <div className="mb-5">
            <label className="block text-sm font-semibold text-[#141b2b] mb-2">
              Reason for rejection <span className="text-[#93000a]">(Required)</span>
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Provide a reason for the rejection..."
              rows={4}
              required
              disabled={actionLoading}
              className="w-full border border-[#bcc9c6] rounded-lg px-3 py-2.5 text-sm text-[#141b2b] placeholder:text-[#6d7a77] resize-none focus:outline-none focus:ring-2 focus:ring-[#00685f]/30 focus:border-[#00685f] transition-colors disabled:bg-gray-50 disabled:cursor-not-allowed"
            />
          </div>

          {/* Actions */}
          <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-2 sm:gap-3">
            <button
              type="button"
              onClick={onCancel}
              disabled={actionLoading}
              className="w-full sm:w-auto border border-[#bcc9c6] bg-white text-[#141b2b] text-sm font-medium px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#f9f9ff] transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!reason.trim() || actionLoading}
              className="w-full sm:w-auto bg-[#93000a] text-white text-sm font-semibold px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#7d000a] transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2 cursor-pointer shadow-sm"
            >
              {actionLoading ? (
                <>
                  <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Rejecting...
                </>
              ) : (
                <>
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path
                      d="M13.5 7.41L12.09 6L7 11.09L1.91 6L0.5 7.41L7 13.91L13.5 7.41Z"
                      fill="white"
                    />
                  </svg>
                  Confirm Rejection
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
