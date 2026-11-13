import UnifiedProfileCard from "../../components/profile/UnifiedProfileCard";

export default function AdminProfilePage() {
  return (
    <div className="p-4 sm:p-6 lg:p-8 max-w-4xl mx-auto flex flex-col gap-6">
      {/* Page Header */}
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-[#00685f]">
          <span>Admin Portal</span>
          <span className="text-[#bcc9c6]">/</span>
          <span>Profile</span>
        </div>
        <h1 className="text-2xl sm:text-3xl font-bold text-[#141b2b] tracking-tight">
          Admin Profile
        </h1>
        <p className="text-sm text-[#565e74]">
          View and update your administrator credentials, personal details, and avatar.
        </p>
      </div>

      {/* Unified Profile Management Card */}
      <UnifiedProfileCard
        title="Administrator Account Information"
        subtitle="Your administrator account details are used across audit logs and administrative events."
      />
    </div>
  );
}
