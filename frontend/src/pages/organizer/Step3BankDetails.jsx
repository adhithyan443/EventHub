import { useState } from "react";
import { Link } from "react-router-dom";
import useOrganizerStore from "../../store/organizerStore";
import { validateIFSC } from "../../utils/validation";

export default function Step3BankDetails({ onNext, onBack, onSaveLater }) {
  const bankName = useOrganizerStore((state) => state.bankName);
  const accountHolderName = useOrganizerStore((state) => state.accountHolderName);
  const accountNumber = useOrganizerStore((state) => state.accountNumber);
  const confirmAccount = useOrganizerStore((state) => state.confirmAccount);
  const ifscCode = useOrganizerStore((state) => state.ifscCode);
  const setField = useOrganizerStore((state) => state.setField);

  const [showAccount, setShowAccount] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [errors, setErrors] = useState({});
  const [touched, setTouched] = useState({});

  function validate(fields = { bankName, accountHolderName, accountNumber, confirmAccount, ifscCode }) {
    const errs = {};

    if (!fields.bankName?.trim()) {
      errs.bankName = "Bank name is required";
    }

    if (!fields.accountHolderName?.trim()) {
      errs.accountHolderName = "Account holder name is required";
    }

    if (!fields.accountNumber?.trim()) {
      errs.accountNumber = "Account number is required";
    } else if (!/^\d{9,18}$/.test(fields.accountNumber.trim())) {
      errs.accountNumber = "Account number must be between 9 and 18 digits";
    }

    if (!fields.confirmAccount?.trim()) {
      errs.confirmAccount = "Please confirm your account number";
    } else if (fields.accountNumber && fields.confirmAccount !== fields.accountNumber) {
      errs.confirmAccount = "Account numbers do not match";
    }

    const ifscErr = validateIFSC(fields.ifscCode);
    if (ifscErr) {
      errs.ifscCode = ifscErr;
    }

    return errs;
  }

  function handleChange(field, value) {
    setField(field, value);
    if (touched[field]) {
      const currentValues = {
        bankName,
        accountHolderName,
        accountNumber,
        confirmAccount,
        ifscCode,
        [field]: value,
      };
      setErrors(validate(currentValues));
    }
  }

  function handleIfscChange(e) {
    const val = e.target.value.toUpperCase();
    handleChange("ifscCode", val);
  }

  function handleBlur(field) {
    setTouched((prev) => ({ ...prev, [field]: true }));
    setErrors(validate());
  }

  function handleSubmit(e) {
    e.preventDefault();
    setTouched({
      bankName: true,
      accountHolderName: true,
      accountNumber: true,
      confirmAccount: true,
      ifscCode: true,
    });

    const validationErrors = validate();
    setErrors(validationErrors);

    if (Object.keys(validationErrors).length === 0) {
      onNext();
    }
  }

  const inputClass = (hasError) =>
    `w-full border rounded-lg px-3.5 py-2.5 text-sm text-ink bg-white outline-none transition-all placeholder:text-slate-400 ${
      hasError
        ? "border-red-500 focus:ring-2 focus:ring-red-400/20"
        : "border-slate-300 focus:border-primary focus:ring-2 focus:ring-primary/20"
    }`;

  const labelClass = "text-sm font-semibold text-slate-800";

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
          Bank Details
        </h1>
        <p className="text-sm sm:text-base text-slate-600">
          Provide your bank account details for organizer payouts.
        </p>
      </div>

      {/* Form card */}
      <form onSubmit={handleSubmit} noValidate className="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-6 sm:p-8">
        <h2 className="font-display text-lg font-bold text-slate-900 pb-4 mb-5 border-b border-slate-100">
          3. Bank Account Details
        </h2>

        {/* Security Banner */}
        <div className="flex items-start sm:items-center gap-2.5 bg-primary/8 border border-primary/20 rounded-xl px-4 py-3 mb-6">
          <div className="text-primary shrink-0 mt-0.5 sm:mt-0">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
              <path d="M7 11V7a5 5 0 0 1 10 0v4" />
            </svg>
          </div>
          <p className="text-xs sm:text-sm text-slate-700 font-medium">
            Your bank information is securely stored and used for organizer settlements.
          </p>
        </div>

        <div className="space-y-5">
          {/* Row 1: Bank Name & Account Holder Name */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-5">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="bankName" className={labelClass}>
                Bank Name <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <input
                id="bankName"
                type="text"
                className={inputClass(errors.bankName)}
                placeholder="e.g. State Bank of India"
                value={bankName}
                onChange={(e) => handleChange("bankName", e.target.value)}
                onBlur={() => handleBlur("bankName")}
                aria-required="true"
                aria-invalid={!!errors.bankName}
                aria-describedby={errors.bankName ? "bankName-error" : undefined}
              />
              {errors.bankName && (
                <p id="bankName-error" className="text-xs text-red-500 mt-0.5">
                  {errors.bankName}
                </p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="accountHolderName" className={labelClass}>
                Account Holder Name <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <input
                id="accountHolderName"
                type="text"
                className={inputClass(errors.accountHolderName)}
                placeholder="Enter account holder name"
                value={accountHolderName}
                onChange={(e) => handleChange("accountHolderName", e.target.value)}
                onBlur={() => handleBlur("accountHolderName")}
                aria-required="true"
                aria-invalid={!!errors.accountHolderName}
                aria-describedby={errors.accountHolderName ? "accountHolder-error" : "holder-help"}
              />
              {errors.accountHolderName ? (
                <p id="accountHolder-error" className="text-xs text-red-500 mt-0.5">
                  {errors.accountHolderName}
                </p>
              ) : (
                <p id="holder-help" className="text-xs text-slate-500">
                  Enter the name exactly as it appears on the bank account.
                </p>
              )}
            </div>
          </div>

          {/* Row 2: Account Number & Confirm Account Number with Show/Hide Toggle */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-5">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="accountNumber" className={labelClass}>
                Account Number <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <div className="relative">
                <input
                  id="accountNumber"
                  type={showAccount ? "text" : "password"}
                  className={`${inputClass(errors.accountNumber)} pr-11 font-mono`}
                  placeholder="Enter account number"
                  value={accountNumber}
                  onChange={(e) => handleChange("accountNumber", e.target.value)}
                  onBlur={() => handleBlur("accountNumber")}
                  aria-required="true"
                  aria-invalid={!!errors.accountNumber}
                  aria-describedby={errors.accountNumber ? "accountNumber-error" : undefined}
                />
                <button
                  type="button"
                  onClick={() => setShowAccount(!showAccount)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 transition-colors p-1"
                  aria-label={showAccount ? "Hide account number" : "Show account number"}
                >
                  {showAccount ? (
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                      <line x1="1" y1="1" x2="23" y2="23" />
                    </svg>
                  ) : (
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                      <circle cx="12" cy="12" r="3" />
                    </svg>
                  )}
                </button>
              </div>
              {errors.accountNumber && (
                <p id="accountNumber-error" className="text-xs text-red-500 mt-0.5">
                  {errors.accountNumber}
                </p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="confirmAccount" className={labelClass}>
                Confirm Account Number <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <div className="relative">
                <input
                  id="confirmAccount"
                  type={showConfirm ? "text" : "password"}
                  className={`${inputClass(errors.confirmAccount)} pr-11 font-mono`}
                  placeholder="Re-enter account number"
                  value={confirmAccount}
                  onChange={(e) => handleChange("confirmAccount", e.target.value)}
                  onBlur={() => handleBlur("confirmAccount")}
                  aria-required="true"
                  aria-invalid={!!errors.confirmAccount}
                  aria-describedby={errors.confirmAccount ? "confirmAccount-error" : undefined}
                />
                <button
                  type="button"
                  onClick={() => setShowConfirm(!showConfirm)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 transition-colors p-1"
                  aria-label={showConfirm ? "Hide confirm account number" : "Show confirm account number"}
                >
                  {showConfirm ? (
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                      <line x1="1" y1="1" x2="23" y2="23" />
                    </svg>
                  ) : (
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                      <circle cx="12" cy="12" r="3" />
                    </svg>
                  )}
                </button>
              </div>
              {errors.confirmAccount && (
                <p id="confirmAccount-error" className="text-xs text-red-500 mt-0.5">
                  {errors.confirmAccount}
                </p>
              )}
            </div>
          </div>

          {/* Row 3: IFSC Code */}
          <div className="flex flex-col gap-1.5 max-w-sm">
            <label htmlFor="ifscCode" className={labelClass}>
              IFSC Code <span className="text-red-500" aria-hidden="true">*</span>
            </label>
            <input
              id="ifscCode"
              type="text"
              maxLength={11}
              className={`${inputClass(errors.ifscCode)} font-mono tracking-wider`}
              placeholder="e.g. SBIN0001234"
              value={ifscCode}
              onChange={handleIfscChange}
              onBlur={() => handleBlur("ifscCode")}
              aria-required="true"
              aria-invalid={!!errors.ifscCode}
              aria-describedby={errors.ifscCode ? "ifsc-error" : "ifsc-help"}
            />
            {errors.ifscCode ? (
              <p id="ifsc-error" className="text-xs text-red-500 mt-0.5">
                {errors.ifscCode}
              </p>
            ) : (
              <p id="ifsc-help" className="text-xs text-slate-500">
                Enter the 11-character IFSC code of your bank branch.
              </p>
            )}
          </div>

          {/* Why do we need this? Callout */}
          <div className="flex items-start gap-3 bg-slate-50 border border-slate-200/80 rounded-xl p-4 mt-6">
            <div className="text-indigo-600 shrink-0 mt-0.5">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <circle cx="12" cy="12" r="10" />
                <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
                <line x1="12" y1="17" x2="12.01" y2="17" />
              </svg>
            </div>
            <div>
              <p className="text-sm font-bold text-slate-800 mb-0.5">
                Why do we need this?
              </p>
              <p className="text-xs sm:text-sm text-slate-600 leading-relaxed">
                These details are required for organizer settlement and future payouts. Your banking information is protected and encrypted at rest.
              </p>
            </div>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex flex-col-reverse sm:flex-row items-center justify-between gap-3 sm:gap-4 mt-8 pt-6 border-t border-slate-100">
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
              className="w-full sm:w-auto h-11 px-6 rounded-lg bg-primary hover:bg-primary/90 text-white text-sm font-semibold transition-colors flex items-center justify-center gap-2 shadow-sm"
            >
              Save & Continue
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
                <path d="M2.5 6H9.5M6.5 3L9.5 6L6.5 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
