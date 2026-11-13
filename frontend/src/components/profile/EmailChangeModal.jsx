import { useState, useEffect, useRef } from "react";
import { requestEmailChange, resendEmailChangeOTP, verifyEmailChange } from "../../api/profileApi";
import notify from "../../utils/toast";

export default function EmailChangeModal({ isOpen, onClose, currentEmail, onSuccess }) {
  const [step, setStep] = useState("request"); // "request" | "verify"
  const [newEmail, setNewEmail] = useState("");
  const [otp, setOtp] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const [resendNotice, setResendNotice] = useState("");

  // Timers
  const [resendCooldown, setResendCooldown] = useState(0);
  const [expiresIn, setExpiresIn] = useState(600); // 10 minutes in seconds

  const cooldownTimerRef = useRef(null);
  const expiryTimerRef = useRef(null);

  // Clear timers on unmount
  useEffect(() => {
    return () => {
      if (cooldownTimerRef.current) clearInterval(cooldownTimerRef.current);
      if (expiryTimerRef.current) clearInterval(expiryTimerRef.current);
    };
  }, []);

  // Reset state when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      setStep("request");
      setNewEmail("");
      setOtp("");
      setErrorMsg("");
      setResendNotice("");
      setResendCooldown(0);
      setExpiresIn(600);
    } else {
      if (cooldownTimerRef.current) clearInterval(cooldownTimerRef.current);
      if (expiryTimerRef.current) clearInterval(expiryTimerRef.current);
    }
  }, [isOpen]);

  // Start cooldown countdown
  const startCooldownTimer = (seconds) => {
    setResendCooldown(seconds);
    if (cooldownTimerRef.current) clearInterval(cooldownTimerRef.current);
    cooldownTimerRef.current = setInterval(() => {
      setResendCooldown((prev) => {
        if (prev <= 1) {
          clearInterval(cooldownTimerRef.current);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  // Start expiry countdown
  const startExpiryTimer = (seconds) => {
    setExpiresIn(seconds);
    if (expiryTimerRef.current) clearInterval(expiryTimerRef.current);
    expiryTimerRef.current = setInterval(() => {
      setExpiresIn((prev) => {
        if (prev <= 1) {
          clearInterval(expiryTimerRef.current);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  const handleRequestSubmit = async (e) => {
    e.preventDefault();
    setErrorMsg("");
    const trimmed = newEmail.trim().toLowerCase();

    if (!trimmed) {
      setErrorMsg("Please enter a new email address.");
      return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(trimmed)) {
      setErrorMsg("Please enter a valid email address.");
      return;
    }

    if (trimmed === (currentEmail || "").trim().toLowerCase()) {
      setErrorMsg("New email must be different from your current email.");
      return;
    }

    setIsLoading(true);
    try {
      const resp = await requestEmailChange(trimmed);
      const data = resp?.data || resp;
      setStep("verify");
      startCooldownTimer(data?.resend_cooldown_seconds || 60);
      startExpiryTimer(data?.expires_in_seconds || 600);
      notify.success("Verification code sent to " + trimmed);
    } catch (err) {
      const msg = err.response?.data?.message || err.message || "Failed to request email change.";
      setErrorMsg(msg);
    } finally {
      setIsLoading(false);
    }
  };

  const handleResend = async () => {
    if (resendCooldown > 0 || isLoading) return;
    setErrorMsg("");
    setResendNotice("");
    setIsLoading(true);

    try {
      const resp = await resendEmailChangeOTP(newEmail.trim().toLowerCase());
      const data = resp?.data || resp;
      startCooldownTimer(data?.resend_cooldown_seconds || 60);
      notify.success("A new verification code was sent.");
    } catch (err) {
      const status = err.response?.status;
      const msg = err.response?.data?.message || err.message || "Failed to resend code.";
      if (status === 429) {
        setErrorMsg(msg);
      } else {
        setResendNotice("Could not activate a new code. If you received an email, your previous code remains valid until expiry.");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleVerifySubmit = async (e) => {
    e.preventDefault();
    setErrorMsg("");
    setResendNotice("");
    const cleanOtp = otp.trim();

    if (!cleanOtp || cleanOtp.length !== 6) {
      setErrorMsg("Please enter the 6-digit verification code.");
      return;
    }

    if (expiresIn <= 0) {
      setErrorMsg("Verification code has expired. Please request a new code.");
      return;
    }

    setIsLoading(true);
    try {
      const resp = await verifyEmailChange({
        newEmail: newEmail.trim().toLowerCase(),
        otp: cleanOtp,
      });
      const data = resp?.data || resp;
      notify.success("Email address updated successfully!");
      if (onSuccess) onSuccess(data);
      onClose();
    } catch (err) {
      const msg = err.response?.data?.message || err.message || "Invalid or expired verification code.";
      setErrorMsg(msg);
    } finally {
      setIsLoading(false);
    }
  };

  if (!isOpen) return null;

  const formatTime = (secs) => {
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${m}:${s < 10 ? "0" : ""}${s}`;
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs animate-in fade-in duration-200">
      <div
        className="w-full max-w-md bg-white rounded-2xl border border-slate-200 shadow-2xl p-6 sm:p-7 relative flex flex-col gap-5 text-slate-800"
        role="dialog"
        aria-modal="true"
        aria-labelledby="email-change-title"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-100 pb-4">
          <div>
            <h3 id="email-change-title" className="text-lg font-bold text-slate-900">
              {step === "request" ? "Change Email Address" : "Verify New Email"}
            </h3>
            <p className="text-xs text-slate-500 mt-0.5">
              {step === "request"
                ? "Enter your new email address to receive a verification code."
                : `Enter the 6-digit code sent to ${newEmail}`}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 transition-colors disabled:opacity-50"
            aria-label="Close modal"
          >
            <svg className="size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Global Error Banner */}
        {errorMsg && (
          <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-xs text-red-700 flex items-start gap-2">
            <svg className="size-4 text-red-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span className="flex-1">{errorMsg}</span>
          </div>
        )}

        {/* Resend Safety Notice */}
        {resendNotice && (
          <div className="p-3 rounded-lg bg-amber-50 border border-amber-200 text-xs text-amber-800 flex items-start gap-2">
            <svg className="size-4 text-amber-600 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span className="flex-1">{resendNotice}</span>
          </div>
        )}

        {step === "request" ? (
          <form onSubmit={handleRequestSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-700">Current Email</label>
              <input
                type="text"
                disabled
                value={currentEmail || ""}
                className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-2.5 text-sm text-slate-500 cursor-not-allowed"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-700">
                New Email Address <span className="text-red-500">*</span>
              </label>
              <input
                type="email"
                required
                autoFocus
                placeholder="e.g. yourname@example.com"
                value={newEmail}
                onChange={(e) => setNewEmail(e.target.value)}
                disabled={isLoading}
                className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all placeholder:text-slate-400"
              />
            </div>

            <div className="flex items-center justify-end gap-2.5 pt-2">
              <button
                type="button"
                onClick={onClose}
                disabled={isLoading}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-600 hover:bg-slate-100 transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-xl bg-primary hover:bg-primary-dark text-white text-xs font-semibold shadow-xs transition-colors disabled:opacity-50 cursor-pointer"
              >
                {isLoading ? (
                  <>
                    <svg className="animate-spin size-3.5 text-white" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                    </svg>
                    <span>Sending Code...</span>
                  </>
                ) : (
                  <span>Send Verification Code</span>
                )}
              </button>
            </div>
          </form>
        ) : (
          <form onSubmit={handleVerifySubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between text-xs">
                <label className="font-semibold text-slate-700">
                  6-Digit Verification Code <span className="text-red-500">*</span>
                </label>
                <span className={`font-mono text-xs ${expiresIn < 60 ? "text-red-600 font-bold" : "text-slate-500"}`}>
                  Expires: {formatTime(expiresIn)}
                </span>
              </div>
              <input
                type="text"
                required
                autoFocus
                maxLength={6}
                pattern="[0-9]{6}"
                placeholder="123456"
                value={otp}
                onChange={(e) => setOtp(e.target.value.replace(/\D/g, ""))}
                disabled={isLoading}
                className="w-full text-center tracking-[0.5em] font-mono text-xl font-bold rounded-xl border border-slate-200 bg-white px-3.5 py-3 text-slate-900 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all placeholder:tracking-normal placeholder:font-normal placeholder:text-sm placeholder:text-slate-400"
              />
            </div>

            <div className="flex items-center justify-between text-xs pt-1">
              <button
                type="button"
                onClick={() => {
                  setStep("request");
                  setErrorMsg("");
                  setResendNotice("");
                }}
                disabled={isLoading}
                className="text-primary hover:underline font-medium"
              >
                Change email address
              </button>

              <button
                type="button"
                onClick={handleResend}
                disabled={resendCooldown > 0 || isLoading}
                className="font-medium text-slate-600 hover:text-slate-900 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {resendCooldown > 0 ? `Resend code in ${resendCooldown}s` : "Resend code"}
              </button>
            </div>

            <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-100">
              <button
                type="button"
                onClick={onClose}
                disabled={isLoading}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-600 hover:bg-slate-100 transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isLoading || otp.length !== 6 || expiresIn <= 0}
                className="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-xl bg-primary hover:bg-primary-dark text-white text-xs font-semibold shadow-xs transition-colors disabled:opacity-50 cursor-pointer"
              >
                {isLoading ? (
                  <>
                    <svg className="animate-spin size-3.5 text-white" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                    </svg>
                    <span>Verifying...</span>
                  </>
                ) : (
                  <span>Verify & Change Email</span>
                )}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
