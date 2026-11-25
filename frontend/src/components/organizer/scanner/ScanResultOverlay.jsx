import { useEffect, useRef, useCallback } from "react";

export default function ScanResultOverlay({
  result,
  onScanNext,
  autoResumeSeconds = 0,
}) {
  const scanNextButtonRef = useRef(null);
  const lastResultKeyRef = useRef(null);
  const isActivatingRef = useRef(false);

  // Reset activation lock whenever result changes
  useEffect(() => {
    isActivatingRef.current = false;
  }, [result]);

  const isSuccess = result?.status === "SUCCESS";
  const isConflict = result?.status === "CONFLICT";
  const isBookingQr = result?.status === "BOOKING_QR";
  const isBadRequest = result?.status === "BAD_REQUEST";
  const isUnauthorized = result?.status === "UNAUTHORIZED";
  const isNotFound = result?.status === "NOT_FOUND";

  // Format timestamp helper handling invalid dates safely
  const formatTime = (isoString) => {
    if (!isoString) return "Just now";
    try {
      const d = isoString instanceof Date ? isoString : new Date(isoString);
      if (Number.isNaN(d.getTime())) {
        return "Just now";
      }
      return d.toLocaleTimeString("en-IN", {
        hour: "numeric",
        minute: "2-digit",
        second: "2-digit",
        hour12: true,
      });
    } catch {
      return "Just now";
    }
  };

  const handleScanNextClick = useCallback(() => {
    if (isActivatingRef.current) return;
    isActivatingRef.current = true;
    onScanNext?.();
  }, [onScanNext]);

  // Keyboard shortcut: Space or Enter triggers Scan Next
  // Scoped to avoid firing when typing into input, textarea, select or other interactive controls
  useEffect(() => {
    if (!result) return;

    const handleKeyDown = (e) => {
      if (e.key !== "Enter" && e.key !== " ") return;
      if (e.repeat) return;

      const target = e.target;
      const tagName = target?.tagName?.toUpperCase();
      const isInteractive =
        tagName === "INPUT" ||
        tagName === "TEXTAREA" ||
        tagName === "SELECT" ||
        (tagName === "BUTTON" && target !== scanNextButtonRef.current) ||
        Boolean(target?.isContentEditable);

      if (isInteractive) return;

      e.preventDefault();
      handleScanNextClick();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [result, handleScanNextClick]);

  // Focus management: Focus Scan Next button only when a NEW result appears
  useEffect(() => {
    if (!result) return;

    const currentKey =
      result.data?.ticket_id ||
      result.data?.ticket_code ||
      result.timestamp ||
      result.status;

    if (lastResultKeyRef.current !== currentKey) {
      lastResultKeyRef.current = currentKey;
      const timer = setTimeout(() => {
        scanNextButtonRef.current?.focus();
      }, 50);
      return () => clearTimeout(timer);
    }
  }, [result]);

  if (!result) return null;

  return (
    <div
      role={isSuccess ? "status" : "alert"}
      aria-live={isSuccess ? "polite" : "assertive"}
      aria-atomic="true"
      className="w-full max-w-md mx-auto my-3 animate-fade-in"
    >
      <div
        className={`rounded-2xl p-5 border-2 shadow-sm transition-all ${
          isSuccess
            ? "bg-[#00685f]/10 border-[#00685f] text-[#00685f]"
            : isBookingQr
            ? "bg-amber-50 border-amber-400 text-amber-900"
            : "bg-[#ba1a1a]/10 border-[#ba1a1a] text-[#ba1a1a]"
        }`}
      >
        <div className="flex items-start gap-3.5">
          {/* Status Icon */}
          <div
            className={`size-10 rounded-full flex items-center justify-center shrink-0 ${
              isSuccess
                ? "bg-[#00685f] text-white"
                : isBookingQr
                ? "bg-amber-500 text-white"
                : "bg-[#ba1a1a] text-white"
            }`}
          >
            {isSuccess ? (
              <svg className="size-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
              </svg>
            ) : isBookingQr ? (
              <svg className="size-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            ) : (
              <svg className="size-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M6 18L18 6M6 6l12 12" />
              </svg>
            )}
          </div>

          {/* Details */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-base font-bold tracking-tight">
                {isSuccess && "Ticket Checked In"}
                {isConflict && "Already Used or Invalid"}
                {isBookingQr && "Booking Pass Detected"}
                {isBadRequest && "Invalid QR Payload"}
                {isUnauthorized && "Not Authorized"}
                {isNotFound && "Ticket Not Found"}
                {!isSuccess && !isConflict && !isBookingQr && !isBadRequest && !isUnauthorized && !isNotFound && (result.title || "Scan Failed")}
              </h3>
              <span
                className={`text-[11px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider ${
                  isSuccess
                    ? "bg-[#00685f] text-white"
                    : isBookingQr
                    ? "bg-amber-200 text-amber-900"
                    : "bg-[#ba1a1a] text-white"
                }`}
              >
                {isSuccess ? "VERIFIED" : isBookingQr ? "ATTENTION" : "REJECTED"}
              </span>
            </div>

            {/* Success Details */}
            {isSuccess && (
              <div className="mt-2.5 pt-2.5 border-t border-[#00685f]/20 flex flex-col gap-1.5 text-xs text-[#141b2b]">
                <div className="flex items-center justify-between">
                  <span className="text-[#565e74] font-medium">Ticket Code:</span>
                  <span className="font-mono font-bold text-sm tracking-wider text-[#00685f]">
                    {result.data?.ticket_code || "—"}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[#565e74] font-medium">Checked In At:</span>
                  <span className="font-semibold">
                    {formatTime(result.data?.used_at)}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[#565e74] font-medium">Ticket ID:</span>
                  <span className="font-mono text-[11px] text-[#565e74] truncate max-w-[200px]" title={result.data?.ticket_id}>
                    {result.data?.ticket_id || "—"}
                  </span>
                </div>
              </div>
            )}

            {/* Error / Warning Details */}
            {!isSuccess && (
              <p className="mt-1 text-xs text-[#141b2b] leading-relaxed">
                {result.message}
              </p>
            )}

            {isBookingQr && (
              <p className="mt-1.5 text-[11px] text-amber-900/90 bg-amber-100/70 p-2 rounded-lg leading-snug">
                The attendee scanned a multi-ticket booking code. Please ask them to open and display their specific <strong>Individual Ticket</strong> QR code.
              </p>
            )}
          </div>
        </div>

        {/* Scan Next Action Button */}
        <div className="mt-4 pt-3 border-t border-black/10 flex items-center justify-between gap-3">
          <span className="text-[11px] text-[#565e74] hidden sm:inline">
            Press <kbd className="px-1.5 py-0.5 bg-white border border-[#bcc9c6] rounded text-[10px] font-mono">Enter</kbd> to continue
          </span>
          <button
            ref={scanNextButtonRef}
            type="button"
            onClick={handleScanNextClick}
            aria-label={
              isSuccess
                ? "Scan next ticket (or press Enter or Space)"
                : "Dismiss result and scan next ticket (or press Enter or Space)"
            }
            className={`w-full sm:w-auto px-5 py-2.5 rounded-xl font-bold text-xs sm:text-sm text-white transition-all shadow-sm flex items-center justify-center gap-2 cursor-pointer ${
              isSuccess
                ? "bg-[#00685f] hover:bg-[#005550]"
                : "bg-[#141b2b] hover:bg-[#2b354f]"
            }`}
          >
            <svg className="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span>Scan Next Ticket</span>
          </button>
        </div>
      </div>
    </div>
  );
}
