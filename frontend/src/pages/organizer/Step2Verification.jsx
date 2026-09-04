import { useState, useRef } from "react";
import { Link } from "react-router-dom";
import useOrganizerStore from "../../store/organizerStore";
import { validatePAN, validateGST } from "../../utils/validation";

const MAX_DOC_SIZE_BYTES = 10 * 1024 * 1024; // 10MB

export default function Step2Verification({ onNext, onBack, onSaveLater }) {
  const panNumber = useOrganizerStore((state) => state.panNumber);
  const gstNumber = useOrganizerStore((state) => state.gstNumber);
  const verificationDocument = useOrganizerStore((state) => state.verificationDocument);
  const verificationDocumentUrl = useOrganizerStore((state) => state.verificationDocumentUrl);
  const businessLogoPreview = useOrganizerStore((state) => state.businessLogoPreview);
  const businessLogoUrl = useOrganizerStore((state) => state.businessLogoUrl);
  const setField = useOrganizerStore((state) => state.setField);
  const setVerificationDocument = useOrganizerStore((state) => state.setVerificationDocument);
  const removeVerificationDocument = useOrganizerStore((state) => state.removeVerificationDocument);
  const setBusinessLogo = useOrganizerStore((state) => state.setBusinessLogo);
  const removeBusinessLogo = useOrganizerStore((state) => state.removeBusinessLogo);

  const [errors, setErrors] = useState({});
  const [touched, setTouched] = useState({});
  const [isDragging, setIsDragging] = useState(false);
  const [docUploadError, setDocUploadError] = useState("");

  const docInputRef = useRef(null);
  const logoInputRef = useRef(null);

  function validate(fields = { panNumber, gstNumber }) {
    const errs = {};

    const panErr = validatePAN(fields.panNumber);
    if (panErr) errs.panNumber = panErr;

    if (fields.gstNumber?.trim()) {
      const gstErr = validateGST(fields.gstNumber);
      if (gstErr) errs.gstNumber = gstErr;
    }

    return errs;
  }

  function handlePanChange(e) {
    const val = e.target.value.toUpperCase();
    setField("panNumber", val);
    if (touched.panNumber) {
      setErrors(validate({ panNumber: val, gstNumber }));
    }
  }

  function handleGstChange(e) {
    const val = e.target.value.toUpperCase();
    setField("gstNumber", val);
    if (touched.gstNumber) {
      setErrors(validate({ panNumber, gstNumber: val }));
    }
  }

  function handleFileSelected(file) {
    setDocUploadError("");
    if (!file) return;

    // Type validation
    const validExtensions = [".pdf", ".jpg", ".jpeg", ".png"];
    const fileExt = file.name.slice(file.name.lastIndexOf(".")).toLowerCase();
    if (!validExtensions.includes(fileExt)) {
      setDocUploadError("Unsupported file type. Please upload a .pdf, .jpg, or .png file.");
      return;
    }

    // Size validation (10MB)
    if (file.size > MAX_DOC_SIZE_BYTES) {
      setDocUploadError("File exceeds the 10MB size limit. Please upload a smaller file.");
      return;
    }

    const sizeInMB = (file.size / (1024 * 1024)).toFixed(1);
    setVerificationDocument({
      name: file.name,
      size: `${sizeInMB} MB`,
      type: fileExt === ".pdf" ? "PDF" : "IMAGE",
      file,
    });
  }

  function handleDrop(e) {
    e.preventDefault();
    setIsDragging(false);
    const file = e.dataTransfer.files?.[0];
    if (file) {
      handleFileSelected(file);
    }
  }

  function handleDragOver(e) {
    e.preventDefault();
    setIsDragging(true);
  }

  function handleDragLeave(e) {
    e.preventDefault();
    setIsDragging(false);
  }

  function handleLogoChange(e) {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!file.type.startsWith("image/")) {
      alert("Please select a valid image file.");
      return;
    }

    const previewUrl = URL.createObjectURL(file);
    setBusinessLogo(file, previewUrl);
  }

  function handleSubmit(e) {
    e.preventDefault();
    setTouched({ panNumber: true, gstNumber: true });

    const validationErrors = validate();
    setErrors(validationErrors);

    if (Object.keys(validationErrors).length === 0) {
      onNext();
    }
  }

  const inputClass = (hasError) =>
    `w-full border rounded-lg px-3.5 py-2.5 text-sm text-ink bg-white outline-none transition-all placeholder:text-slate-400 font-mono tracking-wider ${
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
          Verify your business
        </h1>
        <p className="text-sm sm:text-base text-slate-600">
          Provide your verification details so we can review your organizer application.
        </p>
      </div>

      {/* Form card */}
      <form onSubmit={handleSubmit} noValidate className="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-6 sm:p-8">
        <h2 className="font-display text-lg font-bold text-slate-900 pb-4 mb-6 border-b border-slate-100">
          2. Verification Details
        </h2>

        <div className="space-y-6">
          {/* PAN Number */}
          <div className="flex flex-col gap-1.5">
            <label htmlFor="panNumber" className={labelClass}>
              PAN Number <span className="text-red-500" aria-hidden="true">*</span>
            </label>
            <input
              id="panNumber"
              type="text"
              maxLength={10}
              className={inputClass(errors.panNumber)}
              placeholder="e.g. ABCDE1234F"
              value={panNumber}
              onChange={handlePanChange}
              onBlur={() => {
                setTouched((prev) => ({ ...prev, panNumber: true }));
                setErrors(validate({ panNumber, gstNumber }));
              }}
              aria-required="true"
              aria-invalid={!!errors.panNumber}
              aria-describedby={errors.panNumber ? "pan-error" : "pan-help"}
            />
            {errors.panNumber ? (
              <p id="pan-error" className="text-xs text-red-500 mt-0.5">
                {errors.panNumber}
              </p>
            ) : (
              <p id="pan-help" className="text-xs text-slate-500">
                Enter the PAN associated with your organization or business.
              </p>
            )}
          </div>

          {/* GST Number */}
          <div className="flex flex-col gap-1.5">
            <label htmlFor="gstNumber" className={labelClass}>
              GST Number <span className="text-xs font-normal text-slate-400">(Optional)</span>
            </label>
            <input
              id="gstNumber"
              type="text"
              maxLength={15}
              className={inputClass(errors.gstNumber)}
              placeholder="e.g. 33ABCDE1234F1Z5"
              value={gstNumber}
              onChange={handleGstChange}
              onBlur={() => {
                setTouched((prev) => ({ ...prev, gstNumber: true }));
                setErrors(validate({ panNumber, gstNumber }));
              }}
              aria-invalid={!!errors.gstNumber}
              aria-describedby={errors.gstNumber ? "gst-error" : "gst-help"}
            />
            {errors.gstNumber ? (
              <p id="gst-error" className="text-xs text-red-500 mt-0.5">
                {errors.gstNumber}
              </p>
            ) : (
              <p id="gst-help" className="text-xs text-slate-500">
                GST number is optional if not applicable.
              </p>
            )}
          </div>

          {/* Verification Document Upload */}
          <div className="flex flex-col gap-2">
            <label className={labelClass}>
              Verification Document <span className="text-xs font-normal text-slate-400">(Optional for draft)</span>
            </label>

            {verificationDocument ? (
              <div className="border border-slate-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-slate-50/50">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center text-xs font-bold shrink-0 ${
                    verificationDocument.type === "PDF"
                      ? "bg-red-50 text-red-600 border border-red-200"
                      : "bg-blue-50 text-blue-600 border border-blue-200"
                  }`}>
                    {verificationDocument.type === "PDF" ? "PDF" : "IMG"}
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-slate-800 truncate">
                      {verificationDocument.name}
                    </p>
                    <p className="text-xs text-slate-500">{verificationDocument.size}</p>
                  </div>
                </div>

                <div className="flex items-center gap-4 text-xs sm:text-sm font-semibold self-end sm:self-center">
                  <button
                    type="button"
                    onClick={() => docInputRef.current?.click()}
                    className="text-primary hover:underline cursor-pointer"
                  >
                    Replace
                  </button>
                  <button
                    type="button"
                    onClick={() => removeVerificationDocument()}
                    className="text-red-500 hover:underline cursor-pointer"
                  >
                    Remove
                  </button>
                </div>
              </div>
            ) : (
              <div
                onDrop={handleDrop}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onClick={() => docInputRef.current?.click()}
                className={`border-2 border-dashed rounded-xl p-6 sm:p-8 flex flex-col items-center justify-center gap-2.5 text-center cursor-pointer transition-colors ${
                  isDragging
                    ? "border-primary bg-primary/5"
                    : "border-slate-300 hover:border-primary/60 bg-slate-50/30"
                }`}
              >
                <div className="w-11 h-11 rounded-full bg-slate-100 flex items-center justify-center text-slate-500">
                  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="17 8 12 3 7 8" />
                    <line x1="12" y1="3" x2="12" y2="15" />
                  </svg>
                </div>
                <div>
                  <span className="text-sm font-semibold text-slate-800">
                    Click to upload
                  </span>{" "}
                  <span className="text-sm text-slate-500">or drag and drop</span>
                </div>
                <p className="text-xs text-slate-400">
                  PDF, JPG, or PNG up to 10MB
                </p>
              </div>
            )}

            <input
              ref={docInputRef}
              type="file"
              className="hidden"
              accept=".pdf,.jpg,.jpeg,.png"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) handleFileSelected(file);
              }}
            />

            {docUploadError && (
              <p className="text-xs text-red-500 mt-1">{docUploadError}</p>
            )}

            {/* Document Storage URL */}
            <div className="flex flex-col gap-1.5 mt-2 bg-slate-50/60 p-3.5 rounded-xl border border-slate-200/80">
              <label htmlFor="verificationDocumentUrl" className="text-xs font-semibold text-slate-700">
                Document URL <span className="text-xs font-normal text-slate-400">(Required for API submission)</span>
              </label>
              <input
                id="verificationDocumentUrl"
                type="url"
                className="w-full border border-slate-300 rounded-lg px-3.5 py-2 text-xs text-ink bg-white outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 placeholder:text-slate-400 font-mono"
                placeholder="https://storage.example.com/documents/verification.pdf"
                value={verificationDocumentUrl}
                onChange={(e) => setField("verificationDocumentUrl", e.target.value)}
              />
              <p className="text-xs text-slate-500">
                The backend requires a hosted document URL. In production, this URL is generated automatically upon uploading to object storage.
              </p>
            </div>
          </div>

          {/* Business Logo Upload */}
          <div className="flex flex-col gap-2 pt-2 border-t border-slate-100">
            <label className={labelClass}>Business Logo</label>
            <p className="text-xs text-slate-500 -mt-1">
              This logo will be displayed on your organizer profile and event pages.
            </p>

            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4 sm:gap-6 mt-2">
              <div className="w-16 h-16 sm:w-20 sm:h-20 rounded-full bg-slate-100 border border-slate-200 flex items-center justify-center overflow-hidden shrink-0 shadow-inner">
                {businessLogoPreview ? (
                  <img
                    src={businessLogoPreview}
                    alt="Logo Preview"
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-slate-400" aria-hidden="true">
                    <path d="M3 9a2 2 0 0 1 2-2h.93a2 2 0 0 0 1.664-.89l.812-1.22A2 2 0 0 1 10.07 4h3.86a2 2 0 0 1 1.664.89l.812 1.22A2 2 0 0 0 18.07 7H19a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9Z" />
                    <circle cx="12" cy="13" r="3" />
                  </svg>
                )}
              </div>

              <div className="flex flex-col gap-1.5 flex-1">
                <div className="flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => logoInputRef.current?.click()}
                    className="px-4 py-2 text-xs sm:text-sm font-semibold text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors shadow-sm cursor-pointer"
                  >
                    {businessLogoPreview ? "Change Logo" : "Upload Logo"}
                  </button>
                  {businessLogoPreview && (
                    <button
                      type="button"
                      onClick={() => removeBusinessLogo()}
                      className="text-xs font-semibold text-red-500 hover:underline cursor-pointer"
                    >
                      Remove
                    </button>
                  )}
                </div>
                <p className="text-xs text-slate-400">Recommended size: 400×400px</p>
              </div>

              <input
                ref={logoInputRef}
                type="file"
                className="hidden"
                accept="image/*"
                onChange={handleLogoChange}
              />
            </div>

            {/* Logo Storage URL */}
            <div className="flex flex-col gap-1.5 mt-2 bg-slate-50/60 p-3.5 rounded-xl border border-slate-200/80">
              <label htmlFor="businessLogoUrl" className="text-xs font-semibold text-slate-700">
                Logo URL <span className="text-xs font-normal text-slate-400">(Optional)</span>
              </label>
              <input
                id="businessLogoUrl"
                type="url"
                className="w-full border border-slate-300 rounded-lg px-3.5 py-2 text-xs text-ink bg-white outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 placeholder:text-slate-400 font-mono"
                placeholder="https://storage.example.com/logos/business-logo.png"
                value={businessLogoUrl}
                onChange={(e) => setField("businessLogoUrl", e.target.value)}
              />
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
