import { useState } from "react";
import { Link } from "react-router-dom";
import useOrganizerStore from "../../store/organizerStore";
import { validatePhone, validateURL } from "../../utils/validation";

const businessTypeOptions = [
  "Company",
  "Individual",
  "Partnership",
  "LLP",
  "NGO",
  "College",
];

export default function Step1BusinessInfo({ onNext, onSaveLater }) {
  const businessName = useOrganizerStore((state) => state.businessName);
  const businessType = useOrganizerStore((state) => state.businessType);
  const description = useOrganizerStore((state) => state.description);
  const phone = useOrganizerStore((state) => state.phone);
  const website = useOrganizerStore((state) => state.website);
  const addressLine = useOrganizerStore((state) => state.addressLine);
  const city = useOrganizerStore((state) => state.city);
  const state = useOrganizerStore((state) => state.state);
  const country = useOrganizerStore((state) => state.country);
  const postalCode = useOrganizerStore((state) => state.postalCode);
  const setField = useOrganizerStore((state) => state.setField);

  const [errors, setErrors] = useState({});
  const [touched, setTouched] = useState({});

  function validate(
    fields = {
      businessName,
      businessType,
      description,
      phone,
      website,
      addressLine,
      city,
      state,
      country,
      postalCode,
    }
  ) {
    const errs = {};

    if (!fields.businessName?.trim()) {
      errs.businessName = "Business name is required";
    }

    if (!fields.businessType) {
      errs.businessType = "Please select a business type";
    }

    if (!fields.description?.trim()) {
      errs.description = "Business description is required";
    } else if (fields.description.trim().length < 10) {
      errs.description = "Description should be at least 10 characters";
    }

    if (!fields.phone?.trim()) {
      errs.phone = "Business phone number is required";
    } else {
      const phoneErr = validatePhone(fields.phone);
      if (phoneErr) errs.phone = phoneErr;
    }

    if (fields.website?.trim()) {
      const urlErr = validateURL(fields.website);
      if (urlErr) errs.website = urlErr;
    }

    if (!fields.addressLine?.trim()) {
      errs.addressLine = "Address line is required";
    } else if (fields.addressLine.trim().length < 3) {
      errs.addressLine = "Address line must contain at least 3 characters";
    }

    if (!fields.city?.trim()) {
      errs.city = "City is required";
    }

    if (!fields.state?.trim()) {
      errs.state = "State is required";
    }

    if (!fields.country?.trim()) {
      errs.country = "Country is required";
    }

    if (!fields.postalCode?.trim()) {
      errs.postalCode = "Postal code is required";
    }

    return errs;
  }

  function handleChange(field, value) {
    setField(field, value);
    if (touched[field]) {
      const currentValues = {
        businessName,
        businessType,
        description,
        phone,
        website,
        addressLine,
        city,
        state,
        country,
        postalCode,
        [field]: value,
      };
      setErrors(validate(currentValues));
    }
  }

  function handleBlur(field) {
    setTouched((prev) => ({ ...prev, [field]: true }));
    setErrors(validate());
  }

  function handleNext(e) {
    e.preventDefault();
    setTouched({
      businessName: true,
      businessType: true,
      description: true,
      phone: true,
      website: true,
      addressLine: true,
      city: true,
      state: true,
      country: true,
      postalCode: true,
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
          Become an Organizer
        </h1>
        <p className="text-sm sm:text-base text-slate-600">
          Create and manage events on EventHub
        </p>
      </div>

      {/* Form card */}
      <form onSubmit={handleNext} noValidate className="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-6 sm:p-8">
        <h2 className="font-display text-lg font-bold text-slate-900 pb-4 mb-6 border-b border-slate-100">
          1. Business Information
        </h2>

        <div className="space-y-5">
          {/* Row 1: Business Name & Business Type */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-5">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="businessName" className={labelClass}>
                Business Name <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <input
                id="businessName"
                type="text"
                className={inputClass(errors.businessName)}
                placeholder="e.g. Acme Events Pvt Ltd"
                value={businessName}
                onChange={(e) => handleChange("businessName", e.target.value)}
                onBlur={() => handleBlur("businessName")}
                aria-required="true"
                aria-invalid={!!errors.businessName}
                aria-describedby={errors.businessName ? "businessName-error" : undefined}
              />
              {errors.businessName && (
                <p id="businessName-error" className="text-xs text-red-500 mt-0.5">
                  {errors.businessName}
                </p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="businessType" className={labelClass}>
                Business Type <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <div className="relative">
                <select
                  id="businessType"
                  className={`${inputClass(errors.businessType)} appearance-none pr-9 cursor-pointer`}
                  value={businessType}
                  onChange={(e) => handleChange("businessType", e.target.value)}
                  onBlur={() => handleBlur("businessType")}
                  aria-required="true"
                  aria-invalid={!!errors.businessType}
                  aria-describedby={errors.businessType ? "businessType-error" : undefined}
                >
                  <option value="">Select Type</option>
                  {businessTypeOptions.map((type) => (
                    <option key={type} value={type}>
                      {type}
                    </option>
                  ))}
                </select>
                <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-slate-500">
                  <svg width="12" height="7" viewBox="0 0 12 7" fill="none" aria-hidden="true">
                    <path d="M1 1L6 6L11 1" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </div>
              </div>
              {errors.businessType && (
                <p id="businessType-error" className="text-xs text-red-500 mt-0.5">
                  {errors.businessType}
                </p>
              )}
            </div>
          </div>

          {/* Row 2: Description */}
          <div className="flex flex-col gap-1.5">
            <label htmlFor="description" className={labelClass}>
              Description <span className="text-red-500" aria-hidden="true">*</span>
            </label>
            <textarea
              id="description"
              rows={4}
              className={`${inputClass(errors.description)} resize-none`}
              placeholder="Tell us about the events you organize, audience, past experience..."
              value={description}
              onChange={(e) => handleChange("description", e.target.value)}
              onBlur={() => handleBlur("description")}
              aria-required="true"
              aria-invalid={!!errors.description}
              aria-describedby={errors.description ? "description-error" : undefined}
            />
            {errors.description && (
              <p id="description-error" className="text-xs text-red-500 mt-0.5">
                {errors.description}
              </p>
            )}
          </div>

          {/* Row 3: Business Phone & Website */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-5">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="phone" className={labelClass}>
                Business Phone <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <input
                id="phone"
                type="tel"
                className={inputClass(errors.phone)}
                placeholder="10-digit mobile or phone"
                value={phone}
                onChange={(e) => handleChange("phone", e.target.value)}
                onBlur={() => handleBlur("phone")}
                aria-required="true"
                aria-invalid={!!errors.phone}
                aria-describedby={errors.phone ? "phone-error" : undefined}
              />
              {errors.phone ? (
                <p id="phone-error" className="text-xs text-red-500 mt-0.5">
                  {errors.phone}
                </p>
              ) : (
                <p className="text-xs text-slate-400">Enter a 10-digit contact number</p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="website" className={labelClass}>
                Website <span className="text-xs font-normal text-slate-400">(Optional)</span>
              </label>
              <input
                id="website"
                type="url"
                className={inputClass(errors.website)}
                placeholder="https://example.com"
                value={website}
                onChange={(e) => handleChange("website", e.target.value)}
                onBlur={() => handleBlur("website")}
                aria-invalid={!!errors.website}
                aria-describedby={errors.website ? "website-error" : undefined}
              />
              {errors.website && (
                <p id="website-error" className="text-xs text-red-500 mt-0.5">
                  {errors.website}
                </p>
              )}
            </div>
          </div>

          {/* Section: Business Address */}
          <div className="pt-6 border-t border-slate-100 space-y-5">
            <div>
              <h3 className="text-sm font-bold text-slate-900 mb-1">
                Business Address
              </h3>
              <p className="text-xs text-slate-500">
                Provide your registered business or organization address.
              </p>
            </div>

            {/* Address Line */}
            <div className="flex flex-col gap-1.5">
              <label htmlFor="addressLine" className={labelClass}>
                Address Line <span className="text-red-500" aria-hidden="true">*</span>
              </label>
              <input
                id="addressLine"
                type="text"
                className={inputClass(errors.addressLine)}
                placeholder="Street address, building, suite, unit"
                value={addressLine}
                onChange={(e) => handleChange("addressLine", e.target.value)}
                onBlur={() => handleBlur("addressLine")}
                aria-required="true"
                aria-invalid={!!errors.addressLine}
                aria-describedby={errors.addressLine ? "addressLine-error" : undefined}
              />
              {errors.addressLine && (
                <p id="addressLine-error" className="text-xs text-red-500 mt-0.5">
                  {errors.addressLine}
                </p>
              )}
            </div>

            {/* City & State */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-5">
              <div className="flex flex-col gap-1.5">
                <label htmlFor="city" className={labelClass}>
                  City <span className="text-red-500" aria-hidden="true">*</span>
                </label>
                <input
                  id="city"
                  type="text"
                  className={inputClass(errors.city)}
                  placeholder="e.g. Mumbai"
                  value={city}
                  onChange={(e) => handleChange("city", e.target.value)}
                  onBlur={() => handleBlur("city")}
                  aria-required="true"
                  aria-invalid={!!errors.city}
                  aria-describedby={errors.city ? "city-error" : undefined}
                />
                {errors.city && (
                  <p id="city-error" className="text-xs text-red-500 mt-0.5">
                    {errors.city}
                  </p>
                )}
              </div>

              <div className="flex flex-col gap-1.5">
                <label htmlFor="state" className={labelClass}>
                  State / Province <span className="text-red-500" aria-hidden="true">*</span>
                </label>
                <input
                  id="state"
                  type="text"
                  className={inputClass(errors.state)}
                  placeholder="e.g. Maharashtra"
                  value={state}
                  onChange={(e) => handleChange("state", e.target.value)}
                  onBlur={() => handleBlur("state")}
                  aria-required="true"
                  aria-invalid={!!errors.state}
                  aria-describedby={errors.state ? "state-error" : undefined}
                />
                {errors.state && (
                  <p id="state-error" className="text-xs text-red-500 mt-0.5">
                    {errors.state}
                  </p>
                )}
              </div>
            </div>

            {/* Country & Postal Code */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-5">
              <div className="flex flex-col gap-1.5">
                <label htmlFor="country" className={labelClass}>
                  Country <span className="text-red-500" aria-hidden="true">*</span>
                </label>
                <input
                  id="country"
                  type="text"
                  className={inputClass(errors.country)}
                  placeholder="e.g. India"
                  value={country}
                  onChange={(e) => handleChange("country", e.target.value)}
                  onBlur={() => handleBlur("country")}
                  aria-required="true"
                  aria-invalid={!!errors.country}
                  aria-describedby={errors.country ? "country-error" : undefined}
                />
                {errors.country && (
                  <p id="country-error" className="text-xs text-red-500 mt-0.5">
                    {errors.country}
                  </p>
                )}
              </div>

              <div className="flex flex-col gap-1.5">
                <label htmlFor="postalCode" className={labelClass}>
                  Postal Code / PIN <span className="text-red-500" aria-hidden="true">*</span>
                </label>
                <input
                  id="postalCode"
                  type="text"
                  className={inputClass(errors.postalCode)}
                  placeholder="e.g. 400001"
                  value={postalCode}
                  onChange={(e) => handleChange("postalCode", e.target.value)}
                  onBlur={() => handleBlur("postalCode")}
                  aria-required="true"
                  aria-invalid={!!errors.postalCode}
                  aria-describedby={errors.postalCode ? "postalCode-error" : undefined}
                />
                {errors.postalCode && (
                  <p id="postalCode-error" className="text-xs text-red-500 mt-0.5">
                    {errors.postalCode}
                  </p>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-3 sm:gap-4 mt-8 pt-6 border-t border-slate-100">
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
            Next Step
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M2.5 6H9.5M6.5 3L9.5 6L6.5 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </button>
        </div>
      </form>
    </div>
  );
}
