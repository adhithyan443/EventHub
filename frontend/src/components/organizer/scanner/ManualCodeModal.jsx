import { useState, useEffect, useRef, useCallback } from "react";
import {
  lookupManualBookingCheckIn,
  confirmManualBookingCheckIn,
} from "../../../api/organizerApi";

export default function ManualCodeModal({
  isOpen,
  onClose,
  eventId,
  onCheckInSuccess,
  // Backwards compatibility prop if parent still passes onSubmit
  onSubmit,
}) {
  const [step, setStep] = useState("INPUT"); // "INPUT" | "CONFIRM"
  const [referenceInput, setReferenceInput] = useState("");
  const [validationError, setValidationError] = useState("");

  const [isLoadingLookup, setIsLoadingLookup] = useState(false);
  const [lookupData, setLookupData] = useState(null);

  const [isSubmittingConfirm, setIsSubmittingConfirm] = useState(false);
  const [confirmError, setConfirmError] = useState("");

  const inputRef = useRef(null);
  const dialogRef = useRef(null);
  const previousActiveElementRef = useRef(null);

  // Clean close handler that resets modal state
  const handleModalClose = useCallback(() => {
    setStep("INPUT");
    setReferenceInput("");
    setValidationError("");
    setLookupData(null);
    setConfirmError("");
    setIsLoadingLookup(false);
    setIsSubmittingConfirm(false);
    onClose?.();
  }, [onClose]);

  // Synchronize focus with DOM when modal opens/closes
  useEffect(() => {
    let focusTimer = null;

    if (isOpen) {
      previousActiveElementRef.current = document.activeElement;
      focusTimer = setTimeout(() => {
        inputRef.current?.focus();
      }, 50);
    } else if (previousActiveElementRef.current) {
      previousActiveElementRef.current.focus?.();
      previousActiveElementRef.current = null;
    }

    return () => {
      if (focusTimer) clearTimeout(focusTimer);
    };
  }, [isOpen]);

  // Ensure focus is restored if the modal unmounts while open
  useEffect(() => {
    return () => {
      if (previousActiveElementRef.current) {
        previousActiveElementRef.current.focus?.();
      }
    };
  }, []);

  // Keyboard navigation: Escape key to close & Tab focus trapping
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e) => {
      if (e.key === "Escape") {
        if (!isLoadingLookup && !isSubmittingConfirm) {
          e.preventDefault();
          handleModalClose();
        }
        return;
      }

      if (e.key === "Tab") {
        const dialog = dialogRef.current;
        if (!dialog) return;

        const focusableElements = dialog.querySelectorAll(
          'button:not([disabled]), textarea:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'
        );
        if (!focusableElements || focusableElements.length === 0) return;

        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        if (e.shiftKey) {
          if (document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
          }
        } else {
          if (document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
          }
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, isLoadingLookup, isSubmittingConfirm, handleModalClose]);

  if (!isOpen) return null;

  const isLocked = isLoadingLookup || isSubmittingConfirm;

  // Handle Step 1: Submit Booking Reference for Lookup
  const handleLookupSubmit = async (e) => {
    e.preventDefault();
    if (isLocked) return;

    const trimmed = referenceInput.trim();
    if (!trimmed) {
      setValidationError("Please enter the Booking Reference Number.");
      return;
    }

    // Support legacy QR payload submission if parent provided onSubmit and user pasted a signed payload
    if (trimmed.includes(".") && typeof onSubmit === "function") {
      try {
        await onSubmit(referenceInput.trim());
      } catch (err) {
        setValidationError(err?.response?.data?.message || err?.message || "QR Check-in failed.");
      }
      return;
    }

    if (!eventId) {
      setValidationError("Please select an active event first.");
      return;
    }

    setValidationError("");
    setIsLoadingLookup(true);

    try {
      const res = await lookupManualBookingCheckIn(eventId, trimmed);
      const data = res?.data;
      if (!data) {
        throw new Error("No booking details returned.");
      }
      setLookupData(data);
      setStep("CONFIRM");
    } catch (err) {
      const msg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Booking reference not found or invalid.";
      setValidationError(msg);
    } finally {
      setIsLoadingLookup(false);
    }
  };

  // Handle Step 2: Explicit Confirmation of Check-in
  const handleConfirmSubmit = async () => {
    if (isLocked || !lookupData?.booking_reference || !eventId) return;

    setIsSubmittingConfirm(true);
    setConfirmError("");

    try {
      const res = await confirmManualBookingCheckIn(eventId, lookupData.booking_reference);
      const data = res?.data;
      const message = res?.message || "Booking checked in successfully.";

      onCheckInSuccess?.(data, message);
      handleModalClose();
    } catch (err) {
      const msg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Failed to confirm check-in. The tickets may have already been checked in.";
      setConfirmError(msg);
    } finally {
      setIsSubmittingConfirm(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-fade-in"
      onClick={() => {
        if (!isLocked) handleModalClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="manual-code-modal-title"
      aria-describedby="manual-code-modal-description"
    >
      <div
        ref={dialogRef}
        className={`w-full ${
          step === "CONFIRM" ? "max-w-lg" : "max-w-md"
        } bg-white rounded-2xl shadow-xl border border-[#bcc9c6]/50 p-6 flex flex-col gap-4 max-h-[90vh] overflow-y-auto transition-all`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between pb-3 border-b border-[#bcc9c6]/30">
          <div className="flex items-center gap-2.5">
            <div className="size-9 rounded-lg bg-[#00685f]/10 text-[#00685f] flex items-center justify-center shrink-0">
              <svg className="size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
                />
              </svg>
            </div>
            <div>
              <h3 id="manual-code-modal-title" className="text-base font-bold text-[#141b2b]">
                {step === "CONFIRM" ? "Review & Confirm Check-in" : "Manual Check-in"}
              </h3>
              <p id="manual-code-modal-description" className="text-xs text-[#565e74]">
                {step === "CONFIRM"
                  ? "Verify attendee and ticket details before gate admission"
                  : "Enter the Booking Reference Number from the customer pass"}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={handleModalClose}
            disabled={isLocked}
            aria-label="Close dialog"
            className="p-1 rounded-lg text-[#565e74] hover:bg-gray-100 transition-colors disabled:opacity-50 cursor-pointer"
          >
            <svg className="size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* STEP 1: Enter Reference */}
        {step === "INPUT" && (
          <form onSubmit={handleLookupSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="booking-reference-input" className="text-xs font-semibold text-[#141b2b]">
                Booking Reference Number <span className="text-[#ba1a1a]">*</span>
              </label>
              <input
                id="booking-reference-input"
                ref={inputRef}
                type="text"
                value={referenceInput}
                onChange={(e) => {
                  setReferenceInput(e.target.value);
                  if (validationError) setValidationError("");
                }}
                placeholder="e.g. EVH-949d362c-266"
                className="w-full font-mono text-sm p-3 rounded-xl border border-[#bcc9c6]/70 focus:border-[#00685f] focus:ring-1 focus:ring-[#00685f]/30 outline-none text-[#141b2b] placeholder-[#565e74]/60 tracking-wider transition-all"
                disabled={isLocked}
                autoComplete="off"
                spellCheck="false"
              />
              {validationError ? (
                <span className="text-xs text-[#ba1a1a] font-medium mt-1">{validationError}</span>
              ) : (
                <span className="text-[11px] text-[#565e74] mt-1">
                  Format: <code>EVH-&lt;ID&gt;</code> displayed directly below the barcode on ticket passes
                </span>
              )}
            </div>

            {/* Action buttons */}
            <div className="flex items-center justify-end gap-2.5 pt-2">
              <button
                type="button"
                onClick={handleModalClose}
                disabled={isLocked}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-[#3d4947] hover:bg-gray-100 border border-[#bcc9c6]/60 transition-colors cursor-pointer disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isLocked || !referenceInput.trim()}
                className="px-5 py-2 rounded-xl text-xs font-bold text-white bg-[#00685f] hover:bg-[#005550] transition-colors shadow-sm cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isLoadingLookup ? (
                  <>
                    <span className="size-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    <span>Looking up...</span>
                  </>
                ) : (
                  <span>Verify Reference</span>
                )}
              </button>
            </div>
          </form>
        )}

        {/* STEP 2: Review & Confirm */}
        {step === "CONFIRM" && lookupData && (
          <div className="flex flex-col gap-4">
            {/* Event & Booking Meta */}
            <div className="p-3.5 bg-slate-50 border border-slate-200/80 rounded-xl flex flex-col gap-2">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span className="text-xs font-bold text-[#141b2b] truncate max-w-[280px]">
                  {lookupData.event_title || "Event Admission"}
                </span>
                <code className="font-mono bg-[#eef1ff] px-2 py-0.5 rounded text-xs font-bold text-[#141b2b] border border-slate-200">
                  {lookupData.booking_reference}
                </code>
              </div>

              {/* Customer Contact Badges (Privacy-Preserved) */}
              <div className="pt-2 border-t border-slate-200/60 flex items-center justify-between gap-2 text-xs flex-wrap">
                <div className="flex items-center gap-1.5 text-[#141b2b] font-medium">
                  <svg className="size-3.5 text-[#565e74]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                  <span>{lookupData.customer?.full_name || "Guest Attendee"}</span>
                </div>
                <div className="flex items-center gap-2 text-[11px] text-[#565e74]">
                  {lookupData.customer?.email && <span>{lookupData.customer.email}</span>}
                  {lookupData.customer?.phone && (
                    <>
                      <span>•</span>
                      <span>{lookupData.customer.phone}</span>
                    </>
                  )}
                </div>
              </div>
            </div>

            {/* Admission Summary Stats */}
            <div className="grid grid-cols-3 gap-2 text-center">
              <div className="p-2.5 bg-gray-50 border border-gray-200/70 rounded-xl">
                <span className="text-[10px] uppercase font-bold text-[#565e74] block">Total</span>
                <span className="text-base font-bold text-[#141b2b]">{lookupData.total_admissions}</span>
              </div>
              <div className="p-2.5 bg-gray-50 border border-gray-200/70 rounded-xl">
                <span className="text-[10px] uppercase font-bold text-[#565e74] block">Already Used</span>
                <span className="text-base font-bold text-[#565e74]">{lookupData.already_used_admissions}</span>
              </div>
              <div className="p-2.5 bg-[#00685f]/10 border border-[#00685f]/20 rounded-xl">
                <span className="text-[10px] uppercase font-bold text-[#00685f] block">Ready for Entry</span>
                <span className="text-base font-bold text-[#00685f]">{lookupData.remaining_eligible_admissions}</span>
              </div>
            </div>

            {/* Notice Callout for Partially Used or Ineligible Bookings */}
            {lookupData.already_used_admissions > 0 && lookupData.is_eligible && (
              <div className="p-3 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-900 flex items-start gap-2">
                <svg className="size-4 text-amber-600 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span>
                  <strong>Partial Check-in:</strong> {lookupData.already_used_admissions} of {lookupData.total_admissions} admissions were previously checked in. Confirming will admit the remaining {lookupData.remaining_eligible_admissions} attendee(s).
                </span>
              </div>
            )}

            {!lookupData.is_eligible && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-900 flex items-start gap-2">
                <svg className="size-4 text-red-600 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>
                  <strong>Cannot Check In:</strong> {lookupData.ineligible_reason || "This booking is not eligible for admission."}
                </span>
              </div>
            )}

            {/* Ticket Roster */}
            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-bold text-[#141b2b]">Ticket Breakdown</span>
              <div className="max-h-40 overflow-y-auto divide-y divide-slate-100 border border-slate-200/80 rounded-xl">
                {lookupData.tickets?.map((t, idx) => (
                  <div key={t.ticket_id || idx} className="p-2.5 flex items-center justify-between gap-2 text-xs bg-white">
                    <div className="flex flex-col min-w-0">
                      <div className="flex items-center gap-1.5">
                        <span className="font-semibold text-[#141b2b] truncate">{t.ticket_type_name || "Admission"}</span>
                        {t.seat_label && (
                          <span className="text-[10px] bg-slate-100 text-slate-700 px-1.5 py-0.5 rounded font-mono">
                            {t.seat_label}
                          </span>
                        )}
                      </div>
                      <span className="text-[10px] text-[#565e74] font-mono">{t.ticket_code}</span>
                    </div>

                    <div>
                      {t.status === "VALID" ? (
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-[#00685f]/10 text-[#00685f]">
                          VALID
                        </span>
                      ) : t.status === "USED" ? (
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-gray-100 text-[#565e74]">
                          USED
                        </span>
                      ) : (
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-red-100 text-red-700">
                          {t.status}
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Error Banner on Confirmation Failure */}
            {confirmError && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-700 font-medium">
                {confirmError}
              </div>
            )}

            {/* Actions */}
            <div className="flex items-center justify-end gap-2.5 pt-2 border-t border-[#bcc9c6]/30">
              <button
                type="button"
                onClick={() => setStep("INPUT")}
                disabled={isLocked}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-[#3d4947] hover:bg-gray-100 border border-[#bcc9c6]/60 transition-colors cursor-pointer disabled:opacity-50"
              >
                Back
              </button>
              <button
                type="button"
                onClick={handleConfirmSubmit}
                disabled={isLocked || !lookupData.is_eligible || lookupData.remaining_eligible_admissions <= 0}
                className="px-5 py-2 rounded-xl text-xs font-bold text-white bg-[#00685f] hover:bg-[#005550] transition-colors shadow-sm cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isSubmittingConfirm ? (
                  <>
                    <span className="size-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    <span>Checking In...</span>
                  </>
                ) : (
                  <span>
                    Confirm & Check In ({lookupData.remaining_eligible_admissions})
                  </span>
                )}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
