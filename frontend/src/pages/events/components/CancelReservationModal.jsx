import { useState, useEffect, useRef } from "react";
import { cancelReservation } from "../../../api/reservationApi";
import {
  CloseIcon,
  AlertTriangleIcon,
  CheckCircleIcon,
  ClockIcon,
  CalendarIcon,
  LocationPinIcon,
  TicketIconOutline,
  InfoCircleIcon,
} from "../../bookings/components/BookingIcons";

/**
 * Format date string into a user-friendly format (e.g., "Sat, 14 Nov 2026")
 */
const formatEventDate = (dateStr) => {
  if (!dateStr) return "Date TBA";
  const d = new Date(dateStr);
  if (Number.isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString("en-IN", {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
  });
};

/**
 * Format venue or online location string
 */
const formatEventLocation = (event) => {
  if (!event) return "Location TBA";
  if (event.event_type === "ONLINE") return "Online Event";
  if (event.venue) {
    const parts = [event.venue.name, event.venue.city, event.venue.state].filter(Boolean);
    if (parts.length > 0) return parts.join(", ");
    return event.venue.name || "Venue TBA";
  }
  return "Venue TBA";
};

/**
 * Format seconds into mm:ss display
 */
const formatSeconds = (secs) => {
  if (secs <= 0) return "00:00";
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
};

/**
 * Production-level CancelReservationModal
 * Reusable modal for customer-side pre-payment ticket & seat hold cancellation.
 */
export default function CancelReservationModal({
  isOpen = true,
  reservation,
  event,
  selectedTickets = [],
  selectedSeats = [],
  selectedSeatObjects = [],
  totalAmount = "₹0",
  onClose,
  onSuccess,
  onConfirmCancel,
}) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState(null);
  const [result, setResult] = useState(null);

  // Expiry state calculation
  const expiresAtRaw = reservation?.ExpiresAt || reservation?.expires_at;
  const initialTimeRemaining = () => {
    if (!expiresAtRaw) return 0;
    const expiryTime = new Date(expiresAtRaw).getTime();
    if (Number.isNaN(expiryTime)) return 0;
    return Math.max(0, Math.floor((expiryTime - Date.now()) / 1000));
  };

  const [timeLeft, setTimeLeft] = useState(initialTimeRemaining);
  const isExpired = timeLeft <= 0;

  // Ref to modal container for focus trap
  const modalRef = useRef(null);
  const cancelBtnRef = useRef(null);

  // Background scroll locking
  useEffect(() => {
    if (!isOpen) return;
    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [isOpen]);

  // Escape key handler
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e) => {
      if (e.key === "Escape" && !isSubmitting) {
        if (result) {
          handleFinish();
        } else {
          onClose();
        }
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, isSubmitting, result]);

  // Live countdown timer synchronization
  useEffect(() => {
    if (!isOpen || !expiresAtRaw) return;

    const interval = setInterval(() => {
      const remaining = initialTimeRemaining();
      setTimeLeft(remaining);
      if (remaining <= 0) {
        clearInterval(interval);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isOpen, expiresAtRaw]);

  // Initial focus management
  useEffect(() => {
    if (isOpen && modalRef.current) {
      // Focus the keep button or modal container
      const keepBtn = modalRef.current.querySelector('[data-autofocus="true"]');
      if (keepBtn) {
        keepBtn.focus();
      } else {
        modalRef.current.focus();
      }
    }
  }, [isOpen]);

  if (!isOpen || !reservation) return null;

  const reservationId = reservation.ID || reservation.id;
  const eventTitle = event?.title || "Event Reservation";
  const eventDateStr = formatEventDate(event?.schedule?.event_date || event?.event_date);
  const locationStr = formatEventLocation(event);

  // Compute normalized ticket items
  const ticketItems = (() => {
    if (Array.isArray(selectedTickets) && selectedTickets.length > 0) {
      return selectedTickets.map((t) => ({
        name: t.name || "Ticket",
        quantity: Number(t.quantity || 1),
        price: Number(t.price || 0),
        lineTotal: Number(t.lineTotal ?? (Number(t.price || 0) * Number(t.quantity || 1))),
      }));
    }
    return [];
  })();

  // Compute normalized seat items
  const seatItems = (() => {
    if (Array.isArray(selectedSeatObjects) && selectedSeatObjects.length > 0) {
      return selectedSeatObjects.map((s) => ({
        label: `Row ${s.rowName}, Seat ${s.seatNumber}`,
        section: s.sectionName || null,
        price: Number(s.price || 0),
      }));
    }
    if (Array.isArray(selectedSeats) && selectedSeats.length > 0) {
      return selectedSeats.map((s) => ({
        label: typeof s === "string" ? `Seat ${s}` : `Seat ${s?.seatNumber || s?.id || ""}`,
        section: null,
        price: 0,
      }));
    }
    return [];
  })();

  const hasSpecificSeats = seatItems.length > 0;
  const hasTicketTiers = ticketItems.length > 0;

  // Formatted total string
  const formattedTotal = typeof totalAmount === "number"
    ? `₹${totalAmount.toLocaleString("en-IN")}`
    : totalAmount;

  const handleConfirmCancel = async () => {
    if (isSubmitting || isExpired) return;

    setIsSubmitting(true);
    setError(null);

    try {
      if (typeof onConfirmCancel === "function") {
        const customRes = await onConfirmCancel();
        setResult(customRes || { success: true });
      } else if (reservationId) {
        const response = await cancelReservation(reservationId);
        setResult(response || { success: true });
      } else {
        setResult({ success: true });
      }
    } catch (err) {
      console.error("[CancelReservationModal] Cancellation failed:", err);
      const backendMsg =
        err?.response?.data?.error ||
        err?.response?.data?.message ||
        err?.message ||
        "Unable to cancel reservation. Please check your internet connection and try again.";
      setError(backendMsg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleFinish = () => {
    if (result && typeof onSuccess === "function") {
      onSuccess(result);
    } else if (isExpired && typeof onSuccess === "function") {
      onSuccess({ expired: true, success: true });
    }
    onClose();
  };

  const handleBackdropClick = (e) => {
    if (e.target === e.currentTarget && !isSubmitting) {
      if (result) {
        handleFinish();
      } else {
        onClose();
      }
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="cancel-reservation-modal-title"
      onClick={handleBackdropClick}
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-200"
    >
      <div
        ref={modalRef}
        tabIndex={-1}
        className="relative w-full max-w-lg rounded-2xl bg-white p-5 sm:p-7 shadow-2xl border border-slate-200 max-h-[92vh] overflow-y-auto focus:outline-none"
      >
        {/* Dismiss / Close Icon */}
        <button
          type="button"
          onClick={result ? handleFinish : onClose}
          disabled={isSubmitting}
          className="absolute right-4 top-4 p-1.5 text-slate-400 hover:text-slate-700 rounded-lg transition-colors cursor-pointer disabled:opacity-40"
          aria-label="Close dialog"
        >
          <CloseIcon className="w-5 h-5" />
        </button>

        {/* ============================================================ */}
        {/* PHASE 1: CONFIRMATION / PROGRESS / ERROR VIEW                */}
        {/* ============================================================ */}
        {!result && (
          <div>
            {/* Modal Header */}
            <div className="flex items-start gap-3.5 mb-5">
              <div
                className={`size-11 rounded-full flex items-center justify-center shrink-0 ${
                  isExpired
                    ? "bg-slate-100 text-slate-600"
                    : "bg-rose-100 text-rose-600"
                }`}
              >
                {isExpired ? (
                  <ClockIcon className="w-6 h-6 text-slate-500" />
                ) : (
                  <AlertTriangleIcon className="w-6 h-6" />
                )}
              </div>
              <div className="min-w-0 pr-6">
                <h3
                  id="cancel-reservation-modal-title"
                  className="font-display font-bold text-lg sm:text-xl text-[#141b2b]"
                >
                  {isExpired ? "Reservation Expired" : "Release Reservation?"}
                </h3>
                <p className="mt-0.5 text-xs sm:text-sm text-[#565e74] truncate font-medium">
                  {eventTitle}
                </p>
                <div className="flex items-center gap-3 mt-1.5 text-xs text-[#565e74] flex-wrap">
                  <span className="inline-flex items-center gap-1">
                    <CalendarIcon className="w-3.5 h-3.5 text-[#00796d]" />
                    {eventDateStr}
                  </span>
                  <span className="inline-flex items-center gap-1 truncate max-w-[200px]">
                    <LocationPinIcon className="w-3.5 h-3.5 text-[#00796d]" />
                    {locationStr}
                  </span>
                </div>
              </div>
            </div>

            {/* Countdown / Expiration Status Banner */}
            <div
              className={`rounded-xl border p-3.5 mb-4 text-xs sm:text-sm flex items-center justify-between gap-3 ${
                isExpired
                  ? "border-slate-200 bg-slate-50 text-slate-700"
                  : timeLeft <= 60
                  ? "border-amber-300 bg-amber-50 text-amber-900 animate-pulse"
                  : "border-[#bcc9c6]/60 bg-[#f1f3ff] text-[#141b2b]"
              }`}
            >
              <div className="flex items-center gap-2">
                <ClockIcon
                  className={`w-4 h-4 ${
                    isExpired
                      ? "text-slate-500"
                      : timeLeft <= 60
                      ? "text-amber-600"
                      : "text-[#00796d]"
                  }`}
                />
                <span className="font-semibold">
                  {isExpired ? "Hold status:" : "Hold time remaining:"}
                </span>
              </div>
              <div className="text-right">
                {isExpired ? (
                  <span className="font-bold text-rose-700 uppercase tracking-wide">
                    Expired
                  </span>
                ) : (
                  <strong
                    className={`font-mono text-sm sm:text-base font-bold ${
                      timeLeft <= 60 ? "text-amber-700" : "text-[#00796d]"
                    }`}
                  >
                    {formatSeconds(timeLeft)}
                  </strong>
                )}
              </div>
            </div>

            {/* Cancellation Error Alert if request failed */}
            {error && (
              <div className="mb-4 rounded-xl border border-red-200 bg-red-50 p-3.5 text-xs text-red-700 flex items-start gap-2.5">
                <span className="font-bold text-red-600 text-sm leading-none mt-0.5">⚠</span>
                <div className="flex-1">
                  <strong className="block font-semibold">Unable to Release Hold</strong>
                  <span>{error}</span>
                </div>
              </div>
            )}

            {/* Expired Advisory Banner */}
            {isExpired && (
              <div className="mb-5 rounded-xl border border-slate-200 bg-slate-50 p-3.5 text-xs text-slate-700 flex items-start gap-2.5">
                <InfoCircleIcon className="w-4 h-4 text-slate-500 shrink-0 mt-0.5" />
                <p>
                  Your reservation hold has expired. The held seats or tickets have already been
                  automatically released back to open event inventory.
                </p>
              </div>
            )}

            {/* Selected Tickets & Seats Card */}
            <div className="rounded-xl bg-[#f8fafc] border border-slate-200 p-4 space-y-3 text-xs sm:text-sm mb-5">
              <div className="border-b border-slate-200/80 pb-2">
                <span className="font-semibold text-slate-900 block mb-1">
                  Reserved Items
                </span>
                {/* Reserved Seats List */}
                {hasSpecificSeats && (
                  <div className="space-y-1.5 mt-2">
                    {seatItems.map((s, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between text-slate-700 bg-white px-3 py-1.5 rounded-lg border border-slate-200/60"
                      >
                        <span className="font-medium text-ink flex items-center gap-1.5">
                          <span className="text-[#00796d]">▱</span>
                          {s.label}
                          {s.section && (
                            <span className="text-[11px] text-slate-400">({s.section})</span>
                          )}
                        </span>
                        {s.price > 0 && (
                          <span className="font-semibold text-slate-800">
                            ₹{s.price.toLocaleString("en-IN")}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}

                {/* Ticket Tiers List */}
                {!hasSpecificSeats && hasTicketTiers && (
                  <div className="space-y-1.5 mt-2">
                    {ticketItems.map((t, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between text-slate-700 bg-white px-3 py-2 rounded-lg border border-slate-200/60"
                      >
                        <div>
                          <strong className="block text-ink font-semibold">
                            {t.quantity}x {t.name}
                          </strong>
                          <span className="text-[11px] text-slate-400">
                            ₹{t.price.toLocaleString("en-IN")} each
                          </span>
                        </div>
                        <strong className="font-bold text-slate-900 font-display">
                          ₹{t.lineTotal.toLocaleString("en-IN")}
                        </strong>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Total Reservation Amount */}
              <div className="flex items-center justify-between pt-1">
                <span className="text-[#565e74] font-medium">Total Hold Amount</span>
                <div className="text-right">
                  <strong className="font-display text-base sm:text-lg font-bold text-slate-900">
                    {formattedTotal}
                  </strong>
                  <span className="block text-[11px] font-semibold text-emerald-700">
                    ✓ You have not been charged yet
                  </span>
                </div>
              </div>
            </div>

            {/* Warning / Consequence Callout */}
            {!isExpired && (
              <div className="rounded-xl border border-amber-200 bg-amber-50/70 p-3.5 text-xs text-amber-950 mb-6 space-y-1.5">
                <p className="font-semibold flex items-center gap-1.5 text-amber-900">
                  <span>⚠</span>
                  <span>Consequences of Releasing:</span>
                </p>
                <ul className="list-disc list-inside space-y-1 text-amber-800 text-[11px] sm:text-xs leading-relaxed">
                  <li>
                    Held tickets or seats will be <strong>released immediately</strong> to the public
                    and may be booked by other attendees.
                  </li>
                  <li>
                    No payment was captured; your card or bank account has <strong>not</strong> been
                    charged.
                  </li>
                  <li>
                    This temporary hold cannot be restored once released.
                  </li>
                </ul>
              </div>
            )}

            {/* Modal Actions */}
            <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-3 pt-1">
              <button
                type="button"
                data-autofocus="true"
                onClick={onClose}
                disabled={isSubmitting}
                className="w-full sm:w-auto px-4 py-2.5 rounded-lg border border-[#dfe5e3] bg-white text-xs sm:text-sm font-semibold text-[#565e74] hover:bg-slate-50 transition-colors cursor-pointer disabled:opacity-50"
              >
                {isExpired ? "Close" : "Keep Reservation"}
              </button>

              {!isExpired && (
                <button
                  ref={cancelBtnRef}
                  type="button"
                  onClick={handleConfirmCancel}
                  disabled={isSubmitting}
                  className="w-full sm:w-auto px-5 py-2.5 rounded-lg bg-[#ba1a1a] hover:bg-[#93000a] text-white text-xs sm:text-sm font-semibold shadow-xs transition-colors cursor-pointer flex items-center justify-center gap-2 disabled:opacity-60"
                >
                  {isSubmitting ? (
                    <>
                      <div className="size-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                      <span>Releasing Hold...</span>
                    </>
                  ) : (
                    <span>Release &amp; Cancel</span>
                  )}
                </button>
              )}

              {isExpired && (
                <button
                  type="button"
                  onClick={handleFinish}
                  className="w-full sm:w-auto px-5 py-2.5 rounded-lg bg-[#00796d] hover:bg-[#00685f] text-white text-xs sm:text-sm font-semibold shadow-xs transition-colors cursor-pointer"
                >
                  Select Tickets Again
                </button>
              )}
            </div>
          </div>
        )}

        {/* ============================================================ */}
        {/* PHASE 2: CANCELLATION SUCCESSFUL VIEW                        */}
        {/* ============================================================ */}
        {result && (
          <div>
            {/* Header */}
            <div className="flex items-start gap-3.5 mb-5">
              <div className="size-11 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center shrink-0">
                <CheckCircleIcon className="w-6 h-6" />
              </div>
              <div className="min-w-0 pr-6">
                <h3
                  id="cancel-reservation-modal-title"
                  className="font-display font-bold text-lg sm:text-xl text-[#141b2b]"
                >
                  Reservation Released
                </h3>
                <p className="mt-0.5 text-xs sm:text-sm text-[#565e74] truncate">
                  {eventTitle}
                </p>
              </div>
            </div>

            {/* Outcome Card */}
            <div className="rounded-xl bg-[#f8fafc] border border-slate-200 p-4 space-y-3 text-xs sm:text-sm mb-5">
              <div className="flex items-center justify-between">
                <span className="text-[#565e74]">Hold Status</span>
                <span className="rounded-full bg-slate-200 px-2.5 py-0.5 text-xs font-bold text-slate-800 uppercase">
                  RELEASED
                </span>
              </div>
              <div className="flex items-center justify-between border-t border-slate-200/80 pt-2.5">
                <span className="text-[#565e74]">Payment Status</span>
                <span className="font-semibold text-emerald-700">
                  Not Charged (₹0)
                </span>
              </div>
            </div>

            {/* Confirmation Note */}
            <div className="rounded-xl bg-[#eef1ff] p-4 text-xs text-[#3d4947] mb-6 flex items-start gap-2.5">
              <TicketIconOutline className="w-5 h-5 text-[#00685f] shrink-0 mt-0.5" />
              <p>
                Your reserved tickets have been released back to event inventory. You may start
                a new reservation at any time.
              </p>
            </div>

            {/* Action */}
            <div className="flex justify-end pt-1">
              <button
                type="button"
                onClick={handleFinish}
                className="w-full sm:w-auto px-6 py-2.5 rounded-lg bg-[#00685f] hover:bg-[#00544c] text-white text-xs sm:text-sm font-semibold shadow-xs transition-colors cursor-pointer"
              >
                Continue
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
