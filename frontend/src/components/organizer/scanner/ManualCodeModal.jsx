import { useState, useEffect, useRef } from "react";

export default function ManualCodeModal({
  isOpen,
  onClose,
  onSubmit,
  isSubmitting = false,
}) {
  const [payloadInput, setPayloadInput] = useState("");
  const [validationError, setValidationError] = useState("");

  const inputRef = useRef(null);
  const dialogRef = useRef(null);
  const previousActiveElementRef = useRef(null);
  const isSubmittingRef = useRef(isSubmitting);

  useEffect(() => {
    isSubmittingRef.current = isSubmitting;
  }, [isSubmitting]);

  // Capture previously focused element, reset inputs, and focus textarea cleanly
  useEffect(() => {
    let focusTimer = null;

    if (isOpen) {
      previousActiveElementRef.current = document.activeElement;
      setPayloadInput("");
      setValidationError("");
      isSubmittingRef.current = false;

      focusTimer = setTimeout(() => {
        inputRef.current?.focus();
      }, 50);
    } else {
      if (previousActiveElementRef.current) {
        previousActiveElementRef.current.focus?.();
        previousActiveElementRef.current = null;
      }
    }

    return () => {
      if (focusTimer) {
        clearTimeout(focusTimer);
      }
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

  // Keyboard navigation: Escape key to close & Focus trapping
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e) => {
      // Escape to close when not submitting
      if (e.key === "Escape") {
        if (!isSubmitting && !isSubmittingRef.current) {
          e.preventDefault();
          onClose();
        }
        return;
      }

      // Focus trap navigation
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
  }, [isOpen, isSubmitting, onClose]);

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();

    // Synchronous guard against duplicate rapid submissions
    if (isSubmitting || isSubmittingRef.current) {
      return;
    }

    const trimmed = payloadInput.trim();
    if (!trimmed) {
      setValidationError("Please enter or paste the QR payload.");
      return;
    }

    setValidationError("");
    isSubmittingRef.current = true;

    try {
      await onSubmit?.(trimmed);
    } catch (err) {
      const msg =
        err?.response?.data?.message ||
        err?.message ||
        "Check-in submission failed. Please try again.";
      setValidationError(msg);
    } finally {
      isSubmittingRef.current = false;
    }
  };

  const isLocked = isSubmitting || isSubmittingRef.current;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-fade-in"
      onClick={() => {
        if (!isLocked) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="manual-code-modal-title"
      aria-describedby="manual-code-modal-description"
    >
      <div
        ref={dialogRef}
        className="w-full max-w-md bg-white rounded-2xl shadow-xl border border-[#bcc9c6]/50 p-6 flex flex-col gap-4"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between pb-3 border-b border-[#bcc9c6]/30">
          <div className="flex items-center gap-2.5">
            <div className="size-9 rounded-lg bg-[#00685f]/10 text-[#00685f] flex items-center justify-center">
              <svg className="size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </div>
            <div>
              <h3 id="manual-code-modal-title" className="text-base font-bold text-[#141b2b]">
                Manual Check-in
              </h3>
              <p id="manual-code-modal-description" className="text-xs text-[#565e74]">
                Paste or type ticket payload string
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isLocked}
            aria-label="Close dialog"
            className="p-1 rounded-lg text-[#565e74] hover:bg-gray-100 transition-colors disabled:opacity-50 cursor-pointer"
          >
            <svg className="size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <label htmlFor="qr-payload-input" className="text-xs font-semibold text-[#141b2b]">
              Ticket QR Payload <span className="text-[#ba1a1a]">*</span>
            </label>
            <textarea
              id="qr-payload-input"
              ref={inputRef}
              rows={3}
              value={payloadInput}
              onChange={(e) => {
                setPayloadInput(e.target.value);
                if (validationError) setValidationError("");
              }}
              placeholder="e.g. 123e4567-e89b-12d3-a456-426614174000.a1b2c3d4..."
              className="w-full font-mono text-xs p-3 rounded-xl border border-[#bcc9c6]/70 focus:border-[#00685f] focus:ring-1 focus:ring-[#00685f]/30 outline-none text-[#141b2b] placeholder-[#565e74]/60 resize-none transition-all"
              disabled={isLocked}
            />
            {validationError ? (
              <span className="text-xs text-[#ba1a1a] font-medium">{validationError}</span>
            ) : (
              <span className="text-[11px] text-[#565e74]">
                Format: <code>&lt;TICKET_UUID&gt;.&lt;HMAC_SIGNATURE&gt;</code>
              </span>
            )}
          </div>

          {/* Action buttons */}
          <div className="flex items-center justify-end gap-2.5 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isLocked}
              className="px-4 py-2 rounded-xl text-xs font-semibold text-[#3d4947] hover:bg-gray-100 border border-[#bcc9c6]/60 transition-colors cursor-pointer disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLocked || !payloadInput.trim()}
              className="px-5 py-2 rounded-xl text-xs font-bold text-white bg-[#00685f] hover:bg-[#005550] transition-colors shadow-sm cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {isSubmitting ? (
                <>
                  <span className="size-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  <span>Checking In...</span>
                </>
              ) : (
                <span>Validate & Check In</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
