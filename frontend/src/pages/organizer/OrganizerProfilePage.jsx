import { useState } from "react";
import useOrganizerStore from "../../store/organizerStore";

export default function OrganizerProfilePage() {
  const [logoError, setLogoError] = useState(false);

  const profile = useOrganizerStore((state) => state.profile);
  const isProfileLoading = useOrganizerStore((state) => state.isProfileLoading);
  const profileError = useOrganizerStore((state) => state.profileError);

  if (isProfileLoading && !profile) {
    return (
      <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex items-center justify-center min-h-[50vh]">
        <p className="text-sm text-[#565e74]">Loading your organizer profile...</p>
      </div>
    );
  }

  if (profileError && !profile) {
    return (
      <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto">
        <div className="p-4 rounded-lg bg-[#ffdad6]/60 border border-[#ba1a1a]/30 text-[#ba1a1a] text-sm">
          {profileError}
        </div>
      </div>
    );
  }

  const businessName = profile?.business_name || "-";
  const address = profile?.address || {};
  const bank = profile?.bank_account || {};

  return (
    <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex flex-col gap-6">
      {/* Page Header */}
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-[#00685f]">
          <span>Organizer Portal</span>
          <span className="text-[#bcc9c6]">/</span>
          <span>Profile</span>
        </div>
        <h1 className="text-2xl sm:text-3xl font-bold text-[#141b2b] tracking-tight">
          Organizer Profile
        </h1>
        <p className="text-sm text-[#565e74]">
          Manage and review your organization profile, verification, and payout credentials.
        </p>
      </div>

      {/* Profile Header Card */}
      <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col sm:flex-row sm:items-center justify-between gap-5">
        <div className="flex items-center gap-4 sm:gap-5 min-w-0">
          {/* Business Logo */}
          <div className="relative size-16 sm:size-20 rounded-2xl border border-[#bcc9c6]/60 overflow-hidden bg-[#e8eaf6] shrink-0 flex items-center justify-center shadow-xs">
            {!logoError ? (
              <img
                src={profile?.logo_url}
                alt={businessName}

                onError={() => setLogoError(true)}
                className="size-full object-cover"
              />
            ) : (
              <span className="text-2xl sm:text-3xl font-bold text-[#00685f]">
                {businessName.charAt(0).toUpperCase()}
              </span>
            )}
          </div>

          {/* Business Details */}
          <div className="flex flex-col min-w-0">
            <div className="flex flex-wrap items-center gap-2.5">
              <h2 className="text-xl sm:text-2xl font-bold text-[#141b2b] truncate">
                {businessName}
              </h2>
              {/* Status Badge */}
              <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-bold bg-[#dcfce7] text-[#166534] shrink-0">
                <span className="size-1.5 rounded-full bg-[#166534]" />
                {profile?.status || "UNKNOWN"}
              </span>
            </div>
            <p className="text-sm text-[#565e74] mt-0.5">
              {profile?.businessType || "-"} Account
            </p>
            <p className="text-xs text-[#00685f] font-medium mt-1">
              Verified Organizer
            </p>
          </div>
        </div>

        {/* Edit Profile Button (Presentational / UI-Only) */}
        <div className="shrink-0 pt-2 sm:pt-0">
          <button
            type="button"
            className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-[#bcc9c6]/80 hover:border-[#00685f] bg-white hover:bg-[#00685f]/5 text-sm font-semibold text-[#141b2b] hover:text-[#00685f] transition-all cursor-pointer shadow-xs"
          >
            <svg
              className="size-4 text-inherit"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
              />
            </svg>
            <span>Edit Profile</span>
          </button>
        </div>
      </div>

      {/* Information Cards Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Card 1: Business Information */}
        <section className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs flex flex-col justify-between">
          <div>
            <div className="flex items-center gap-2.5 pb-4 mb-4 border-b border-[#bcc9c6]/30">
              <div className="p-2 rounded-lg bg-[#00685f]/10 text-[#00685f]">
                <svg
                  className="size-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
                  />
                </svg>
              </div>
              <div>
                <h3 className="text-base font-bold text-[#141b2b]">
                  Business Information
                </h3>
                <p className="text-xs text-[#565e74]">
                  Basic profile and contact details
                </p>
              </div>
            </div>

            <div className="space-y-3.5 text-sm">
              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Business Name
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {businessName}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Business Type
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {profile?.business_type || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Phone
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {profile?.phone || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Email
                </span>
                <span className="text-[#00685f] font-semibold break-words sm:text-right">
                  {profile?.email || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Website
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {profile?.website}
                </span>
              </div>

              <div className="flex flex-col py-2 border-b border-gray-100 last:border-0 gap-1.5">
                <span className="text-[#565e74] font-medium">
                  Description
                </span>
                <p className="text-[#141b2b] bg-[#f9f9ff] p-3 rounded-xl border border-[#bcc9c6]/30 text-xs sm:text-sm leading-relaxed break-words">
                  {profile?.description}
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* Column 2: Business Address & Verification */}
        <div className="flex flex-col gap-6">
          {/* Card 2: Business Address */}
          <section className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs">
            <div className="flex items-center gap-2.5 pb-4 mb-4 border-b border-[#bcc9c6]/30">
              <div className="p-2 rounded-lg bg-[#00685f]/10 text-[#00685f]">
                <svg
                  className="size-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                </svg>
              </div>
              <div>
                <h3 className="text-base font-bold text-[#141b2b]">
                  Business Address
                </h3>
                <p className="text-xs text-[#565e74]">
                  Registered location of your business
                </p>
              </div>
            </div>

            <div className="space-y-3.5 text-sm">
              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Address
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {address.address_line || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  City
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {address.city || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  State
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {address.state || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Country
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {address.country || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Postal Code
                </span>
                <span className="text-[#141b2b] font-semibold break-words sm:text-right">
                  {address.postal_code || "-"}
                </span>
              </div>
            </div>
          </section>

          {/* Card 3: Verification Information */}
          <section className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs">
            <div className="flex items-center gap-2.5 pb-4 mb-4 border-b border-[#bcc9c6]/30">
              <div className="p-2 rounded-lg bg-[#00685f]/10 text-[#00685f]">
                <svg
                  className="size-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                  />
                </svg>
              </div>
              <div>
                <h3 className="text-base font-bold text-[#141b2b]">
                  Verification Information
                </h3>
                <p className="text-xs text-[#565e74]">
                  Compliance & tax identifiers (masked)
                </p>
              </div>
            </div>

            <div className="space-y-3.5 text-sm">
              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  PAN
                </span>
                <span className="text-[#141b2b] font-semibold font-mono tracking-wider break-words sm:text-right">
                  {profile?.verification_pan || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-start justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  GST
                </span>
                <span className="text-[#141b2b] font-semibold font-mono tracking-wider break-words sm:text-right">
                  {profile?.verification_gst  || "-"}
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-center justify-between py-2 border-b border-gray-100 last:border-0 gap-1 sm:gap-4">
                <span className="text-[#565e74] font-medium shrink-0 sm:w-36">
                  Status
                </span>
                <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-bold bg-emerald-50 text-emerald-700 border border-emerald-200/60">
                  <span className="size-1.5 rounded-full bg-emerald-500" />
                  Verified
                </span>
              </div>
            </div>
          </section>
        </div>
      </div>

      {/* Card 4: Bank Information */}
      <section className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-xs">
        <div className="flex items-center gap-2.5 pb-4 mb-4 border-b border-[#bcc9c6]/30">
          <div className="p-2 rounded-lg bg-[#00685f]/10 text-[#00685f]">
            <svg
              className="size-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M8 14v3m4-3v3m4-3v3M3 21h18M3 10h18M3 7l9-4 9 4M4 10h16v11H4V10z"
              />
            </svg>
          </div>
          <div>
            <h3 className="text-base font-bold text-[#141b2b]">
              Bank Information
            </h3>
            <p className="text-xs text-[#565e74]">
              Payout bank account details for ticket proceeds
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-4 rounded-xl bg-[#f9f9ff] border border-[#bcc9c6]/30">
            <div className="text-xs text-[#565e74] font-medium uppercase tracking-wider mb-1">
              Bank Name
            </div>
            <div className="text-[#141b2b] text-base font-bold truncate">
              {bank.bank_name || "-"}
            </div>
          </div>

          <div className="p-4 rounded-xl bg-[#f9f9ff] border border-[#bcc9c6]/30">
            <div className="text-xs text-[#565e74] font-medium uppercase tracking-wider mb-1">
              Account Holder Name
            </div>
            <div className="text-[#141b2b] text-base font-bold truncate">
              {bank.account_holder_name || "-"}
            </div>
          </div>

          <div className="p-4 rounded-xl bg-[#f9f9ff] border border-[#bcc9c6]/30">
            <div className="text-xs text-[#565e74] font-medium uppercase tracking-wider mb-1">
              Account Number
            </div>
            <div className="text-[#141b2b] text-base font-bold font-mono tracking-wider truncate">
              {bank.account_number_masked || "-"}
            </div>
          </div>

          <div className="p-4 rounded-xl bg-[#f9f9ff] border border-[#bcc9c6]/30">
            <div className="text-xs text-[#565e74] font-medium uppercase tracking-wider mb-1">
              IFSC Code
            </div>
            <div className="text-[#141b2b] text-base font-bold font-mono tracking-wider truncate">
              {bank.ifsc_code || "-"}
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
