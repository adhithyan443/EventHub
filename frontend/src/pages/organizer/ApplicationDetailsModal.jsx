import { useEffect } from "react";
import useOrganizerStore from "../../store/organizerStore";
import { maskPAN, maskAccountNumber } from "../../utils/validation";

function DetailItem({ label, value, isMasked = false }) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-baseline justify-between py-2 border-b border-slate-100 last:border-0 gap-1 sm:gap-4 text-xs sm:text-sm">
      <span className="text-slate-500 font-medium shrink-0 sm:w-40">{label}</span>
      <span className={`text-slate-900 font-semibold break-words sm:text-right ${isMasked ? "font-mono" : ""}`}>
        {value || "—"}
      </span>
    </div>
  );
}

export default function ApplicationDetailsModal({ isOpen, onClose }) {
  const businessName = useOrganizerStore((state) => state.businessName);
  const businessType = useOrganizerStore((state) => state.businessType);
  const description = useOrganizerStore((state) => state.description);
  const phone = useOrganizerStore((state) => state.phone);
  const website = useOrganizerStore((state) => state.website);
  const panNumber = useOrganizerStore((state) => state.panNumber);
  const gstNumber = useOrganizerStore((state) => state.gstNumber);
  const verificationDocument = useOrganizerStore((state) => state.verificationDocument);
  const businessLogoPreview = useOrganizerStore((state) => state.businessLogoPreview);
  const bankName = useOrganizerStore((state) => state.bankName);
  const accountHolder = useOrganizerStore((state) => state.accountHolder);
  const accountNumber = useOrganizerStore((state) => state.accountNumber);
  const ifscCode = useOrganizerStore((state) => state.ifscCode);
  const applicationId = useOrganizerStore((state) => state.applicationId) || "APP001";
  const applicationStatus = useOrganizerStore((state) => state.applicationStatus) || "PENDING";
  const submissionDate = useOrganizerStore((state) => state.submissionDate) || "10 August 2026";

  useEffect(() => {
    function handleKeyDown(e) {
      if (e.key === "Escape") {
        onClose();
      }
    }
    if (isOpen) {
      document.body.style.overflow = "hidden";
      window.addEventListener("keydown", handleKeyDown);
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs animate-fade-in"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-2xl border border-slate-200 shadow-2xl max-w-2xl w-full max-h-[90vh] flex flex-col overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100 shrink-0 bg-slate-50/50">
          <div>
            <h2 id="modal-title" className="font-display text-lg sm:text-xl font-bold text-slate-900">
              Application Details
            </h2>
            <p className="text-xs text-slate-500">ID: {applicationId}</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
            aria-label="Close modal"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-6 overflow-y-auto space-y-6">
          {/* Section 1: Application Information */}
          <div>
            <h3 className="text-xs font-bold uppercase tracking-wider text-primary mb-3">
              Application Information
            </h3>
            <div className="bg-slate-50 rounded-xl p-4 border border-slate-100">
              <DetailItem label="Application ID" value={applicationId} />
              <DetailItem label="Submission Date" value={submissionDate} />
              <div className="flex items-center justify-between py-2 text-xs sm:text-sm">
                <span className="text-slate-500 font-medium">Status</span>
                <span className="px-2.5 py-0.5 rounded-full text-xs font-bold bg-primary/10 text-primary border border-primary/20">
                  {applicationStatus}
                </span>
              </div>
            </div>
          </div>

          {/* Section 2: Business Information */}
          <div>
            <h3 className="text-xs font-bold uppercase tracking-wider text-primary mb-3">
              Business Information
            </h3>
            <div className="bg-slate-50 rounded-xl p-4 border border-slate-100">
              <DetailItem label="Business Name" value={businessName} />
              <DetailItem label="Business Type" value={businessType} />
              <DetailItem label="Description" value={description} />
              <DetailItem label="Phone" value={phone} />
              <DetailItem label="Website" value={website} />
            </div>
          </div>

          {/* Section 3: Verification */}
          <div>
            <h3 className="text-xs font-bold uppercase tracking-wider text-primary mb-3">
              Verification (Masked)
            </h3>
            <div className="bg-slate-50 rounded-xl p-4 border border-slate-100">
              <DetailItem label="PAN Number" value={maskPAN(panNumber)} isMasked />
              <DetailItem label="GST Number" value={gstNumber || "N/A"} isMasked={!!gstNumber} />
              <DetailItem label="Document" value={verificationDocument?.name || "Not uploaded"} />
              <div className="flex items-center justify-between py-2 text-xs sm:text-sm">
                <span className="text-slate-500 font-medium">Business Logo</span>
                {businessLogoPreview ? (
                  <img
                    src={businessLogoPreview}
                    alt="Business logo"
                    className="w-7 h-7 rounded-full object-cover border border-slate-200"
                  />
                ) : (
                  <span className="text-slate-400 font-medium">Default</span>
                )}
              </div>
            </div>
          </div>

          {/* Section 4: Bank Details */}
          <div>
            <h3 className="text-xs font-bold uppercase tracking-wider text-primary mb-3">
              Bank Details (Protected)
            </h3>
            <div className="bg-slate-50 rounded-xl p-4 border border-slate-100">
              <DetailItem label="Bank Name" value={bankName} />
              <DetailItem label="Account Holder" value={accountHolder} />
              <DetailItem label="Account Number" value={maskAccountNumber(accountNumber)} isMasked />
              <DetailItem label="IFSC Code" value={ifscCode} isMasked />
            </div>
          </div>
        </div>

        {/* Modal Footer */}
        <div className="px-6 py-4 border-t border-slate-100 bg-slate-50/50 flex justify-end shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="h-10 px-6 rounded-lg bg-primary hover:bg-primary/90 text-white text-sm font-semibold transition-colors shadow-sm cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
