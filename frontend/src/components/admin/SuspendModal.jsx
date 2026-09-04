import { useState } from "react";

export default function SuspendModal({ organizer, onConfirm, onCancel }) {
  const [reason, setReason] = useState("");

  if (!organizer) return null;

  const handleSubmit = (e) => {
    e.preventDefault();
    if (reason.trim()) {
      onConfirm(organizer.id, reason.trim());
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={onCancel}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[500px] overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <form onSubmit={handleSubmit} className="p-6">
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
                Suspend Organizer?
              </h2>
              <p className="text-[#3d4947] text-sm mt-1 leading-5">
                You are about to suspend{" "}
                <span className="font-semibold text-[#141b2b]">{organizer.name}</span>{" "}
                (<span className="font-mono text-xs">{organizer.id}</span>). They will no
                longer be able to publish events or access their dashboard. Existing ticket
                sales may be affected.
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
              placeholder="Provide a reason for the suspension..."
              rows={4}
              required
              className="w-full border border-[#bcc9c6] rounded-lg px-3 py-2.5 text-sm text-[#141b2b] placeholder:text-[#6d7a77] resize-none focus:outline-none focus:ring-2 focus:ring-[#00685f]/30 focus:border-[#00685f] transition-colors"
            />
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={onCancel}
              className="border border-[#bcc9c6] bg-white text-[#141b2b] text-sm font-medium px-5 py-2 rounded-lg hover:bg-[#f9f9ff] transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!reason.trim()}
              className="bg-[#93000a] text-white text-sm font-semibold px-5 py-2 rounded-lg hover:bg-[#7d000a] transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-2 cursor-pointer shadow-sm"
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path
                  d="M7 0C3.13 0 0 3.13 0 7C0 10.87 3.13 14 7 14C10.87 14 14 10.87 14 7C14 3.13 10.87 0 7 0ZM8.5 10.5H5.5V6.5H8.5V10.5ZM8.5 5H5.5V2H8.5V5Z"
                  fill="white"
                />
              </svg>
              Confirm Suspension
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
