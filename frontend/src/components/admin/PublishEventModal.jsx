import { useState } from "react";
import { publishAdminEvent } from "../../api/adminEventApi";
import notify from "../../utils/toast";

export default function PublishEventModal({
  event,
  isOpen,
  onClose,
  onSuccess,
}) {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  if (!isOpen || !event) return null;

  const handleConfirm = async () => {
    try {
      setSubmitting(true);
      setError("");
      await publishAdminEvent(event.id);
      notify.success("Event published successfully!");
      if (onSuccess) {
        onSuccess(event.id);
      }
      onClose();
    } catch (err) {
      console.error("Failed to publish event:", err);
      const msg = err.response?.data?.message || err.message || "Failed to publish event.";
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-[2px] animate-in fade-in duration-200">
      <div className="bg-white rounded-2xl shadow-2xl border border-[#dce2f7] w-full max-w-md overflow-hidden">
        {/* Header */}
        <div className="p-5 border-b border-[#dce2f7] flex items-center justify-between bg-[#f9f9ff]">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-[#e0f2fe] text-[#0369a1] flex items-center justify-center">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </div>
            <div>
              <h2 className="text-base font-bold text-[#141b2b]">
                Publish Event
              </h2>
              <p className="text-xs text-[#3d4947]">
                Make this event live on the platform
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center text-[#3d4947] hover:bg-[#eef0fb] transition-colors cursor-pointer"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="p-5 space-y-4 text-xs">
          {error && (
            <div className="rounded-lg bg-[#fff5f4] border border-[#ffdad6] text-[#ba1a1a] p-3 font-medium">
              {error}
            </div>
          )}

          <p className="text-[#3d4947] leading-relaxed">
            You are about to publish the draft event:
          </p>

          <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-3 space-y-1">
            <div className="font-bold text-[#141b2b] text-sm">
              {event.title}
            </div>
            <div className="text-[#3d4947]">
              Organizer: <span className="font-medium text-[#141b2b]">{event.organizer_business_name || "Organizer"}</span>
            </div>
          </div>

          <div className="space-y-1.5 text-[#3d4947] bg-[#f0fdf4] border border-[#bbf7d0] p-3 rounded-xl">
            <div className="font-semibold text-[#15803d] mb-1">
              Publication Checklist:
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[#15803d]">✓</span> Organizer must be active
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[#15803d]">✓</span> Category must be active
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[#15803d]">✓</span> Schedule and sales window verified
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[#15803d]">✓</span> Ticket inventory or seated layout configured
            </div>
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
            className="bg-[#00685f] hover:bg-[#005049] text-white text-xs font-semibold px-5 py-2 rounded-lg transition-colors cursor-pointer flex items-center gap-2 shadow-sm disabled:opacity-50"
          >
            {submitting ? "Publishing..." : "Publish Event"}
          </button>
        </div>
      </div>
    </div>
  );
}
