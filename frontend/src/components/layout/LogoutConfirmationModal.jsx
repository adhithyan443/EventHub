import { useEffect } from "react";
import { LogoutIcon } from "./icons";

export default function LogoutConfirmationModal({
  isOpen,
  onClose,
  onConfirm,
  loading = false,
  title = "Log Out",
  message = "Are you sure you want to log out? You will need to sign in again to access your account.",
}) {
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e) => {
      if (e.key === "Escape" && !loading) {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, loading, onClose]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-xs animate-in fade-in duration-150"
      onClick={() => {
        if (!loading) onClose();
      }}
      role="presentation"
    >
      {/* Modal Dialog Card */}
      <div
        className="relative bg-white rounded-2xl shadow-2xl border border-[#bcc9c6]/40 w-full max-w-md overflow-hidden animate-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="logout-dialog-title"
        aria-describedby="logout-dialog-description"
      >
        <div className="p-5 sm:p-6">
          <div className="flex items-start gap-4">
            <div className="size-11 rounded-full bg-red-50 border border-red-100 flex items-center justify-center shrink-0 text-red-600 mt-0.5">
              <LogoutIcon className="size-5 text-red-600" />
            </div>

            <div className="flex-1 min-w-0">
              <h3
                id="logout-dialog-title"
                className="text-lg font-bold text-[#141b2b] leading-6"
              >
                {title}
              </h3>
              <p
                id="logout-dialog-description"
                className="mt-2 text-sm text-[#565e74] leading-relaxed"
              >
                {message}
              </p>
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="bg-[#f9f9ff] px-5 sm:px-6 py-3.5 sm:py-4 border-t border-[#bcc9c6]/30 flex flex-col-reverse sm:flex-row items-center justify-end gap-2 sm:gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="w-full sm:w-auto px-4 py-2 text-xs sm:text-sm font-medium text-[#565e74] hover:text-[#141b2b] bg-white border border-[#bcc9c6] hover:bg-gray-50 rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>

          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            className="w-full sm:w-auto px-4 py-2 text-xs sm:text-sm font-semibold text-white bg-red-600 hover:bg-red-700 rounded-lg transition-colors shadow-xs cursor-pointer flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <>
                <svg
                  className="animate-spin size-4 text-white"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                  />
                </svg>
                <span>Logging out...</span>
              </>
            ) : (
              <>
                <LogoutIcon className="size-4 text-white" />
                <span>Logout</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
