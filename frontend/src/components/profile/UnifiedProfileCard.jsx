import { useState, useEffect, useRef } from "react";
import useAuthStore from "../../store/authStore";
import { updateProfile, uploadAvatar, removeAvatar, getProfile } from "../../api/profileApi";
import EmailChangeModal from "./EmailChangeModal";
import notify from "../../utils/toast";

export default function UnifiedProfileCard({ title = "Personal Information", subtitle = "Manage your personal profile and account credentials." }) {
  const user = useAuthStore((state) => state.user);
  const updateUser = useAuthStore((state) => state.updateUser);
  const revalidateSession = useAuthStore((state) => state.revalidateSession);

  const [form, setForm] = useState({
    fullName: "",
    phone: "",
  });
  const [initialForm, setInitialForm] = useState({
    fullName: "",
    phone: "",
  });

  const [isSaving, setIsSaving] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [isEmailModalOpen, setIsEmailModalOpen] = useState(false);
  const [imageFailed, setImageFailed] = useState(false);
  const hasRetriedRef = useRef(false);
  const fileInputRef = useRef(null);

  // Sync initial form values from authStore user
  useEffect(() => {
    if (user) {
      const name = user.fullName || user.full_name || "";
      const phone = user.phone || "";
      setForm({ fullName: name, phone });
      setInitialForm({ fullName: name, phone });
      setImageFailed(false);
      hasRetriedRef.current = false;
    }
  }, [user]);

  // On mount, fetch fresh profile from backend to resolve fresh presigned avatar
  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const resp = await getProfile();
        const data = resp?.data || resp;
        if (mounted && data) {
          updateUser(data);
        }
      } catch {
        // Silently preserve current session state on transient network failures
      }
    })();
    return () => {
      mounted = false;
    };
  }, []);

  const avatarUrl = user?.profileImage || user?.ProfileImage || user?.avatar || "";

  const handleImageError = async () => {
    if (!hasRetriedRef.current) {
      hasRetriedRef.current = true;
      try {
        const result = await revalidateSession();
        if (result?.user?.profileImage) {
          return;
        }
      } catch {
        // Revalidation failed
      }
    }
    setImageFailed(true);
  };

  const getInitials = (name) => {
    if (!name || typeof name !== "string") return "U";
    const parts = name.trim().split(/\s+/);
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  };

  const isDirty = form.fullName !== initialForm.fullName || form.phone !== initialForm.phone;

  const handleFileChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Reset input value so same file can be selected again
    e.target.value = "";

    // Client-side validations
    const allowedTypes = ["image/jpeg", "image/png", "image/webp"];
    if (!allowedTypes.includes(file.type)) {
      notify.error("Please upload a valid image file (JPEG, PNG, or WebP).");
      return;
    }

    if (file.size > 5 * 1024 * 1024) {
      notify.error("Profile picture must not exceed 5 MB.");
      return;
    }

    setIsUploading(true);
    setImageFailed(false);
    hasRetriedRef.current = false;

    try {
      const resp = await uploadAvatar(file);
      const data = resp?.data || resp;
      updateUser(data);
      notify.success("Profile photo updated successfully!");
    } catch (err) {
      const msg = err.response?.data?.message || err.message || "Failed to upload profile photo.";
      notify.error(msg);
    } finally {
      setIsUploading(false);
    }
  };

  const handleRemovePhoto = async () => {
    if (!window.confirm("Are you sure you want to remove your profile photo?")) return;

    setIsRemoving(true);
    try {
      const resp = await removeAvatar();
      const data = resp?.data || resp;
      updateUser(data);
      setImageFailed(false);
      notify.success("Profile photo removed.");
    } catch (err) {
      const msg = err.response?.data?.message || err.message || "Failed to remove profile photo.";
      notify.error(msg);
    } finally {
      setIsRemoving(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!isDirty || isSaving) return;

    const trimmedName = form.fullName.trim();
    if (trimmedName.length < 2 || trimmedName.length > 100) {
      notify.error("Full name must be between 2 and 100 characters.");
      return;
    }

    const trimmedPhone = form.phone.trim();
    if (trimmedPhone && !/^\+?[1-9]\d{6,14}$/.test(trimmedPhone)) {
      notify.error("Please enter a valid phone number with 7 to 15 digits (e.g. +919876543210).");
      return;
    }

    setIsSaving(true);
    try {
      const resp = await updateProfile({
        full_name: trimmedName,
        phone: trimmedPhone,
      });
      const data = resp?.data || resp;
      updateUser(data);
      setInitialForm({ fullName: trimmedName, phone: trimmedPhone });
      notify.success("Profile information updated successfully!");
    } catch (err) {
      const msg = err.response?.data?.message || err.message || "Failed to update profile.";
      notify.error(msg);
    } finally {
      setIsSaving(false);
    }
  };

  const roleBadgeColors = {
    ADMIN: "bg-purple-100 text-purple-800 border-purple-200",
    ORGANIZER: "bg-emerald-100 text-emerald-800 border-emerald-200",
    CUSTOMER: "bg-blue-100 text-blue-800 border-blue-200",
  };

  const role = (user?.role || "CUSTOMER").toUpperCase();

  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 sm:p-7 shadow-xs flex flex-col gap-6 text-slate-800">
      {/* Header with Title and Role Badge */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 pb-5">
        <div>
          <h2 className="font-display text-xl font-bold text-slate-900 tracking-tight">{title}</h2>
          <p className="text-xs sm:text-sm text-slate-500 mt-0.5">{subtitle}</p>
        </div>
        <span
          className={`self-start sm:self-auto inline-flex items-center px-3 py-1 rounded-full text-xs font-bold border ${
            roleBadgeColors[role] || "bg-slate-100 text-slate-800 border-slate-200"
          }`}
        >
          {role}
        </span>
      </div>

      {/* Avatar Section */}
      <div className="flex flex-col sm:flex-row sm:items-center gap-5 pb-5 border-b border-slate-100">
        <div className="relative group size-20 sm:size-24 rounded-full shrink-0 border-2 border-slate-200 overflow-hidden bg-slate-100 shadow-xs flex items-center justify-center">
          {avatarUrl && !imageFailed ? (
            <img
              src={avatarUrl}
              alt={user?.fullName || "Avatar"}
              onError={handleImageError}
              className="size-full object-cover"
            />
          ) : (
            <span className="font-bold text-xl sm:text-2xl text-slate-600 select-none">
              {getInitials(user?.fullName || user?.full_name)}
            </span>
          )}

          {/* Loading Overlay */}
          {(isUploading || isRemoving) && (
            <div className="absolute inset-0 bg-slate-900/50 flex items-center justify-center backdrop-blur-2xs">
              <svg className="animate-spin size-6 text-white" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
            </div>
          )}
        </div>

        <div className="flex flex-col gap-2.5">
          <div>
            <h3 className="text-sm font-bold text-slate-900">Profile Photo</h3>
            <p className="text-xs text-slate-500 mt-0.5">
              Supports JPEG, PNG, or WebP. Max file size: 5 MB.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
            <input
              type="file"
              ref={fileInputRef}
              onChange={handleFileChange}
              accept="image/jpeg,image/png,image/webp"
              className="hidden"
            />
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={isUploading || isRemoving}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 text-white text-xs font-semibold shadow-xs transition-colors cursor-pointer disabled:opacity-50"
            >
              <svg className="size-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <span>{avatarUrl && !imageFailed ? "Change Photo" : "Upload Photo"}</span>
            </button>

            {avatarUrl && !imageFailed && (
              <button
                type="button"
                onClick={handleRemovePhoto}
                disabled={isUploading || isRemoving}
                className="px-3.5 py-2 rounded-xl border border-red-200 hover:bg-red-50 text-red-600 text-xs font-semibold transition-colors cursor-pointer disabled:opacity-50"
              >
                Remove
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Profile Form */}
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        {/* Full Name */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-semibold text-slate-700">
            Full Name <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            required
            name="fullName"
            value={form.fullName}
            onChange={(e) => setForm((prev) => ({ ...prev, fullName: e.target.value }))}
            placeholder="Your full name"
            className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-800 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all placeholder:text-slate-400"
          />
        </div>

        {/* Email Field with Change Button */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-semibold text-slate-700">Email Address</label>
          <div className="flex items-center gap-2">
            <input
              type="email"
              disabled
              value={user?.email || ""}
              className="flex-1 rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-2.5 text-sm text-slate-500 cursor-not-allowed"
            />
            <button
              type="button"
              onClick={() => setIsEmailModalOpen(true)}
              className="shrink-0 px-4 py-2.5 rounded-xl border border-slate-300 hover:border-slate-400 bg-white hover:bg-slate-50 text-xs font-semibold text-slate-700 transition-colors cursor-pointer shadow-xs"
            >
              Change Email
            </button>
          </div>
          <p className="text-[11px] text-slate-500">
            Changing your email requires verifying a 6-digit OTP sent to the new address.
          </p>
        </div>

        {/* Phone Field */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-semibold text-slate-700">Phone Number</label>
          <input
            type="tel"
            name="phone"
            value={form.phone}
            onChange={(e) => setForm((prev) => ({ ...prev, phone: e.target.value }))}
            placeholder="+919876543210"
            className="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-800 focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all placeholder:text-slate-400"
          />
          <p className="text-[11px] text-slate-500">
            Format: +[country code][number] (e.g. +919876543210).
          </p>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-100">
          {isDirty && (
            <button
              type="button"
              onClick={() => setForm(initialForm)}
              disabled={isSaving}
              className="px-4 py-2 text-xs font-semibold text-slate-500 hover:text-slate-700 transition-colors"
            >
              Reset
            </button>
          )}

          <button
            type="submit"
            disabled={!isDirty || isSaving}
            className="inline-flex items-center justify-center gap-2 px-6 py-2.5 rounded-xl bg-primary hover:bg-primary-dark text-white text-xs font-semibold shadow-xs transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          >
            {isSaving ? (
              <>
                <svg className="animate-spin size-3.5 text-white" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                </svg>
                <span>Saving...</span>
              </>
            ) : (
              <span>Save Changes</span>
            )}
          </button>
        </div>
      </form>

      {/* Email Change Modal */}
      <EmailChangeModal
        isOpen={isEmailModalOpen}
        onClose={() => setIsEmailModalOpen(false)}
        currentEmail={user?.email}
        onSuccess={(updated) => {
          if (updated) updateUser(updated);
        }}
      />
    </div>
  );
}
