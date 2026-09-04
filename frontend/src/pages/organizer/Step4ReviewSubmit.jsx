import { useState } from "react";
import { Link } from "react-router-dom";
import useOrganizerStore from "../../store/organizerStore";
import { maskPAN, maskAccountNumber } from "../../utils/validation";

function ReviewRow({ label, value, isMasked = false }) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-slate-100 last:border-0 gap-1 sm:gap-4 text-xs sm:text-sm">
      <span className="text-slate-500 font-medium shrink-0 sm:w-36">{label}:</span>
      <span className={`text-slate-800 font-semibold break-words sm:text-right ${isMasked ? "font-mono" : ""}`}>
        {value || "—"}
      </span>
    </div>
  );
}

export default function Step4ReviewSubmit({ onSubmit, onBack, onSaveLater, onEditStep }) {
  const businessName = useOrganizerStore((state) => state.businessName);
  const businessType = useOrganizerStore((state) => state.businessType);
  const description = useOrganizerStore((state) => state.description);
  const phone = useOrganizerStore((state) => state.phone);
  const website = useOrganizerStore((state) => state.website);

  const addressLine = useOrganizerStore((state) => state.addressLine);
  const city = useOrganizerStore((state) => state.city);
  const stateVal = useOrganizerStore((state) => state.state);
  const country = useOrganizerStore((state) => state.country);
  const postalCode = useOrganizerStore((state) => state.postalCode);

  const panNumber = useOrganizerStore((state) => state.panNumber);
  const gstNumber = useOrganizerStore((state) => state.gstNumber);
  const verificationDocument = useOrganizerStore((state) => state.verificationDocument);
  const verificationDocumentUrl = useOrganizerStore((state) => state.verificationDocumentUrl);
  const businessLogoPreview = useOrganizerStore((state) => state.businessLogoPreview);
  const businessLogoUrl = useOrganizerStore((state) => state.businessLogoUrl);

  const bankName = useOrganizerStore((state) => state.bankName);
  const accountHolderName = useOrganizerStore((state) => state.accountHolderName);
  const accountNumber = useOrganizerStore((state) => state.accountNumber);
  const ifscCode = useOrganizerStore((state) => state.ifscCode);
  const termsAccepted = useOrganizerStore((state) => state.termsAccepted);
  const setField = useOrganizerStore((state) => state.setField);

  const isSubmitting = useOrganizerStore((state) => state.isSubmitting);
  const submitError = useOrganizerStore((state) => state.submitError);
  const isResubmission = useOrganizerStore((state) => state.isResubmission);
  const rejectionReason = useOrganizerStore((state) => state.rejectionReason);

  const [validationError, setValidationError] = useState("");

  async function handleFormSubmit(e) {
    e.preventDefault();
    setValidationError("");

    if (!termsAccepted || isSubmitting) return;

    if (!verificationDocumentUrl || !verificationDocumentUrl.trim()) {
      setValidationError(
        "Verification document upload is not available yet. Please upload the document through the configured storage service before submitting."
      );
      return;
    }

    try {
      await onSubmit();
    } catch {
      // Step4 will display submitError from the store
    }
  }

  return (
    <div className="w-full">
      {/* Breadcrumb */}
      <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 mb-6 text-sm text-slate-500">
        <Link to="/profile" className="hover:text-primary transition-colors">
          Profile
        </Link>
        <svg width="6" height="10" viewBox="0 0 6 10" fill="none" aria-hidden="true">
          <path d="M1 1L5 5L1 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
        <span className="font-semibold text-slate-900">Become an Organizer</span>
      </nav>

      {/* Page header */}
      <div className="mb-8">
        <h1 className="font-display text-2xl sm:text-3xl font-bold text-slate-900 tracking-tight mb-2">
          Review your application
        </h1>
        <p className="text-sm sm:text-base text-slate-600">
          Review your information carefully before submitting your organizer application.
        </p>
      </div>

      <form onSubmit={handleFormSubmit} className="space-y-6">
        {/* Rejection Feedback Banner on Review Step */}
        {(isResubmission || rejectionReason) && (
          <div className="bg-amber-50/90 border border-amber-200/90 rounded-2xl p-4 sm:p-5 flex items-start gap-3.5 shadow-xs">
            <div className="w-8 h-8 rounded-lg bg-amber-500/15 text-amber-700 flex items-center justify-center shrink-0 mt-0.5">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </div>
            <div className="flex-1">
              <span className="text-[11px] font-bold uppercase tracking-wider text-amber-900 block mb-0.5">
                Admin Rejection Feedback
              </span>
              <p className="text-xs sm:text-sm font-medium text-amber-900">
                {rejectionReason || "Please verify your details and documentation before resubmitting."}
              </p>
              <p className="text-xs text-amber-700 mt-1">
                Ensure that all requested changes have been addressed below before clicking Resubmit Application.
              </p>
            </div>
          </div>
        )}

        {/* 2x2 Review Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          {/* Card 1: Business Information */}
          <div className="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-5 sm:p-6 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-100">
                <div className="flex items-center gap-2 text-primary">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <rect x="2" y="7" width="20" height="14" rx="2" ry="2" />
                    <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
                  </svg>
                  <h3 className="font-display text-base font-bold text-slate-900">
                    Business Info
                  </h3>
                </div>
                <button
                  type="button"
                  onClick={() => onEditStep(1)}
                  className="text-xs font-semibold text-primary hover:underline flex items-center gap-1 cursor-pointer"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                  Edit
                </button>
              </div>

              <div className="space-y-1">
                <ReviewRow label="Business Name" value={businessName} />
                <ReviewRow label="Business Type" value={businessType} />
                <ReviewRow label="Description" value={description} />
                <ReviewRow label="Phone" value={phone} />
                <ReviewRow label="Website" value={website} />
                <ReviewRow label="Address" value={addressLine} />
                <ReviewRow label="City / State" value={city && stateVal ? `${city}, ${stateVal}` : city || stateVal} />
                <ReviewRow label="Postal / Country" value={postalCode && country ? `${postalCode}, ${country}` : postalCode || country} />
              </div>
            </div>
          </div>

          {/* Card 2: Verification Details */}
          <div className="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-5 sm:p-6 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-100">
                <div className="flex items-center gap-2 text-primary">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                    <path d="m9 12 2 2 4-4" />
                  </svg>
                  <h3 className="font-display text-base font-bold text-slate-900">
                    Verification
                  </h3>
                </div>
                <button
                  type="button"
                  onClick={() => onEditStep(2)}
                  className="text-xs font-semibold text-primary hover:underline flex items-center gap-1 cursor-pointer"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                  Edit
                </button>
              </div>

              <div className="space-y-1">
                <ReviewRow label="PAN Number" value={maskPAN(panNumber)} isMasked />
                <ReviewRow label="GST Number" value={gstNumber || "N/A"} isMasked={!!gstNumber} />
                <ReviewRow label="Document File" value={verificationDocument?.name || "Not uploaded"} />
                <ReviewRow label="Document URL" value={verificationDocumentUrl || "Not provided"} />
                <div className="flex items-center justify-between py-2 text-xs sm:text-sm">
                  <span className="text-slate-500 font-medium">Business Logo:</span>
                  {businessLogoPreview ? (
                    <div className="flex items-center gap-2">
                      <img
                        src={businessLogoPreview}
                        alt="Logo thumbnail"
                        className="w-7 h-7 rounded-full object-cover border border-slate-200 shadow-xs"
                      />
                      <span className="text-slate-800 font-semibold">Uploaded</span>
                    </div>
                  ) : (
                    <span className="text-slate-400 font-medium">Default</span>
                  )}
                </div>
                {businessLogoUrl && (
                  <ReviewRow label="Logo URL" value={businessLogoUrl} />
                )}
              </div>
            </div>
          </div>

          {/* Card 3: Bank Details */}
          <div className="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-5 sm:p-6 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-100">
                <div className="flex items-center gap-2 text-primary">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <rect x="2" y="10" width="20" height="12" rx="2" />
                    <path d="M12 2 2 7h20L12 2z" />
                    <path d="M6 10v6M18 10v6M12 10v6" />
                  </svg>
                  <h3 className="font-display text-base font-bold text-slate-900">
                    Bank Details
                  </h3>
                </div>
                <button
                  type="button"
                  onClick={() => onEditStep(3)}
                  className="text-xs font-semibold text-primary hover:underline flex items-center gap-1 cursor-pointer"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                  Edit
                </button>
              </div>

              <div className="space-y-1">
                <ReviewRow label="Bank Name" value={bankName} />
                <ReviewRow label="Account Holder" value={accountHolderName} />
                <ReviewRow label="Account No" value={maskAccountNumber(accountNumber)} isMasked />
                <ReviewRow label="IFSC Code" value={ifscCode} isMasked />
              </div>
            </div>
          </div>

          {/* Card 4: Summary Card */}
          <div className="bg-primary/5 rounded-2xl border border-primary/20 shadow-sm p-5 sm:p-6 flex flex-col justify-between">
            <div>
              <div className="flex items-center gap-2 pb-3 mb-3 border-b border-primary/15 text-primary">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
                  <rect x="8" y="2" width="8" height="4" rx="1" ry="1" />
                  <path d="m9 14 2 2 4-4" />
                </svg>
                <h3 className="font-display text-base font-bold text-slate-900">
                  Summary
                </h3>
              </div>

              <div className="space-y-1 text-xs sm:text-sm">
                <div className="flex items-center justify-between py-2 border-b border-primary/10">
                  <span className="text-slate-500 font-medium">Type:</span>
                  <span className="text-slate-800 font-semibold">Organizer Application</span>
                </div>
                <div className="flex items-center justify-between py-2">
                  <span className="text-slate-500 font-medium">Status:</span>
                  <div className="flex items-center gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-emerald-500" />
                    <span className="text-emerald-700 font-bold">Ready to Submit</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="mt-4 flex items-start gap-2.5 bg-white/80 border border-slate-200/80 rounded-xl p-3">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-slate-500 shrink-0 mt-0.5" aria-hidden="true">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="16" x2="12" y2="12" />
                <line x1="12" y1="8" x2="12.01" y2="8" />
              </svg>
              <p className="text-xs text-slate-600 leading-relaxed">
                After submission, your application will be reviewed by the EventHub administration team.
              </p>
            </div>
          </div>
        </div>

        {/* Consent Checkbox */}
        <div className="bg-white rounded-2xl border border-slate-200/80 p-5 sm:p-6 shadow-sm">
          <label className="flex items-start gap-3.5 cursor-pointer select-none">
            <input
              type="checkbox"
              className="mt-1 w-4 h-4 rounded border-slate-300 text-primary focus:ring-primary/30 cursor-pointer"
              checked={termsAccepted}
              onChange={(e) => setField("termsAccepted", e.target.checked)}
              required
            />
            <span className="text-xs sm:text-sm text-slate-700 leading-relaxed">
              I confirm that all information provided is accurate and I agree to the{" "}
              <Link to="/terms" target="_blank" className="text-primary font-semibold hover:underline">
                EventHub Organizer Terms and Conditions
              </Link>
              .
            </span>
          </label>
        </div>

        {/* Error Alert */}
        {(validationError || submitError) && (
          <div className="p-4 rounded-xl bg-red-50 border border-red-200 text-red-700 text-xs sm:text-sm flex items-start gap-3">
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="shrink-0 mt-0.5 text-red-500"
              aria-hidden="true"
            >
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
            <div className="flex-1 font-medium leading-relaxed">
              {validationError || submitError}
            </div>
          </div>
        )}

        {/* Action buttons */}
        <div className="flex flex-col-reverse sm:flex-row items-center justify-between gap-3 sm:gap-4 pt-2">
          <button
            type="button"
            onClick={onBack}
            className="w-full sm:w-auto h-11 px-5 rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-sm font-semibold transition-colors flex items-center justify-center gap-2 shadow-sm"
          >
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M9.5 6H2.5M5.5 3L2.5 6L5.5 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            Back
          </button>

          <div className="flex flex-col sm:flex-row items-center gap-3 w-full sm:w-auto">
            <button
              type="button"
              onClick={onSaveLater}
              className="w-full sm:w-auto h-11 px-5 rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-sm font-semibold transition-colors shadow-sm"
            >
              Save & Continue Later
            </button>
            <button
              type="submit"
              disabled={!termsAccepted || isSubmitting}
              className={`w-full sm:w-auto h-11 px-6 rounded-lg text-white text-sm font-semibold transition-all flex items-center justify-center gap-2 shadow-sm ${
                termsAccepted && !isSubmitting
                  ? "bg-primary hover:bg-primary/90 cursor-pointer"
                  : "bg-slate-300 cursor-not-allowed opacity-75"
              }`}
            >
              {isSubmitting ? (
                <>
                  <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span>{isResubmission ? "Resubmitting..." : "Submitting..."}</span>
                </>
              ) : (
                <>
                  {isResubmission ? "Resubmit Application" : "Submit Application"}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <line x1="22" y1="2" x2="11" y2="13" />
                    <polygon points="22 2 15 22 11 13 2 9 22 2" />
                  </svg>
                </>
              )}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
