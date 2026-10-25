import { CloseIcon, RefundUndoIcon } from "./BookingIcons";

export default function RefundDetailsModal({ booking, onClose }) {
  if (!booking) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="refund-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-in fade-in duration-200"
    >
      <div className="relative w-full max-w-md rounded-2xl bg-white p-6 shadow-xl border border-slate-200">
        <button
          type="button"
          onClick={onClose}
          className="absolute right-4 top-4 p-1.5 text-slate-400 hover:text-slate-700 rounded-lg transition-colors cursor-pointer"
          aria-label="Close modal"
        >
          <CloseIcon className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-4">
          <div className="h-10 w-10 rounded-full bg-[#ffdad6] text-[#93000a] flex items-center justify-center">
            <RefundUndoIcon className="w-5 h-5" />
          </div>
          <div>
            <h3 id="refund-modal-title" className="font-display font-bold text-lg text-ink">
              Refund Details
            </h3>
            <p className="text-xs text-slate-500">Booking ID: {booking.id}</p>
          </div>
        </div>

        <div className="rounded-xl bg-[#f8fafc] border border-slate-200/80 p-4 space-y-3 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-slate-500">Event</span>
            <span className="font-semibold text-slate-900 text-right truncate max-w-[200px]">
              {booking.title}
            </span>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-slate-500">Refund Status</span>
            <span
              className={`text-xs font-semibold px-2.5 py-0.5 rounded-full ${
                booking.refundStatus === "PROCESSING"
                  ? "bg-amber-100 text-amber-800"
                  : "bg-emerald-100 text-emerald-800"
              }`}
            >
              {booking.refundStatus || "PROCESSING"}
            </span>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-slate-500">Refund Amount</span>
            <span className="font-bold text-slate-900 text-base">
              ₹{Number(booking.refundAmount || 0).toLocaleString("en-IN")}
            </span>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-slate-500">Refund Date</span>
            <span className="text-slate-700 font-medium">
              {booking.refundDate || booking.bookedOn}
            </span>
          </div>

          <div className="flex items-start justify-between gap-4">
            <span className="text-slate-500 shrink-0">Method</span>
            <span className="text-slate-700 font-medium text-right text-xs">
              {booking.refundMethod || "UPI (Original Payment Method)"}
            </span>
          </div>
        </div>

        <p className="mt-4 text-xs text-slate-500 leading-relaxed bg-slate-50 p-3 rounded-lg border border-slate-100">
          {booking.refundNote ||
            "Your refund will be credited back to your original payment method in 3–5 working days."}
        </p>

        <div className="mt-6 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="w-full sm:w-auto px-5 py-2.5 rounded-lg bg-primary hover:bg-primary/90 text-white font-semibold text-sm transition-colors cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
