import { useState } from "react";
import { deleteAdminEvent } from "../../api/adminEventApi";
import notify from "../../utils/toast";

export default function DeleteEventModal({
  event,
  isOpen,
  onClose,
  onSuccess,
}) {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [isConflict, setIsConflict] = useState(false);

  if (!isOpen || !event) return null;

  const handleConfirm = async () => {
    try {
      setSubmitting(true);
      setError("");
      setIsConflict(false);

      await deleteAdminEvent(event.id);
      notify.success("Event permanently deleted");
      if (onSuccess) {
        onSuccess(event.id);
      }
      onClose();
    } catch (err) {
      console.error("Failed to delete event:", err);
      const status = err.response?.status;
      const msg =
        err.response?.data?.message ||
        err.message ||
        "Failed to delete event.";

      if (status === 409 || msg.includes("confirmed bookings") || msg.includes("active reservations")) {
        setIsConflict(true);
        setError(
          "Cannot Delete Event: This event has confirmed bookings or active reservation holds. Events with booking records cannot be deleted to preserve transactional integrity and audit history."
        );
      } else {
        setError(msg);
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-[2px] animate-in fade-in duration-200">
      <div className="bg-white rounded-2xl shadow-2xl border border-[#dce2f7] w-full max-w-md overflow-hidden">
        {/* Header */}
        <div className="p-5 border-b border-[#dce2f7] flex items-center justify-between bg-[#fff5f4]">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-[#fee2e2] text-[#ba1a1a] flex items-center justify-center">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                <line x1="10" y1="11" x2="10" y2="17" />
                <line x1="14" y1="11" x2="14" y2="17" />
              </svg>
            </div>
            <div>
              <h2 className="text-base font-bold text-[#ba1a1a]">
                Delete Event
              </h2>
              <p className="text-xs text-[#3d4947]">
                Permanent removal from platform
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center text-[#3d4947] hover:bg-[#ffdad6] transition-colors cursor-pointer"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="p-5 space-y-4 text-xs">
          {error && (
            <div
              className={`rounded-xl p-3.5 border ${
                isConflict
                  ? "bg-[#fff5f4] border-[#ba1a1a] text-[#ba1a1a]"
                  : "bg-[#fff5f4] border-[#ffdad6] text-[#ba1a1a]"
              } leading-relaxed font-medium`}
            >
              <div className="font-bold flex items-center gap-1.5 mb-1">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="12" />
                  <line x1="12" y1="16" x2="12.01" y2="16" />
                </svg>
                {isConflict ? "Deletion Blocked (409 Conflict)" : "Error"}
              </div>
              {error}
            </div>
          )}

          <p className="text-[#3d4947] leading-relaxed">
            Are you sure you want to permanently delete this event? This action will remove the event record and all associated configuration.
          </p>

          <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-3 space-y-1">
            <div className="font-bold text-[#141b2b] text-sm">
              {event.title}
            </div>
            <div className="text-[#3d4947]">
              Status: <span className="font-semibold text-[#141b2b]">{event.status}</span>
            </div>
            <div className="text-[#3d4947]">
              Organizer: <span className="font-medium text-[#141b2b]">{event.organizer_business_name || "Organizer"}</span>
            </div>
          </div>

          <div className="p-3 bg-[#fef9c3] border border-[#fef08a] rounded-xl text-[#854d0e] leading-relaxed">
            <span className="font-bold block mb-0.5">Integrity Guard:</span>
            Deletion requires <strong>0 confirmed bookings</strong> and <strong>0 active reservations</strong>.
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-[#dce2f7] bg-[#f9f9ff] flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            disabled={submitting}
            className="px-4 py-2 text-xs font-semibold text-[#3d4947] hover:bg-[#eef0fb] rounded-lg transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            disabled={submitting}
            className="bg-[#ba1a1a] hover:bg-[#93000a] text-white text-xs font-semibold px-5 py-2 rounded-lg transition-colors cursor-pointer flex items-center gap-2 shadow-sm disabled:opacity-50"
          >
            {submitting ? "Deleting..." : "Permanently Delete"}
          </button>
        </div>
      </div>
    </div>
  );
}
