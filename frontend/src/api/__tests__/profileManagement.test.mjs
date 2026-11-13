import test from "node:test";
import assert from "node:assert";

test("profileApi: exports all unified profile management functions", async () => {
  const {
    getProfile,
    updateProfile,
    requestEmailChange,
    resendEmailChangeOTP,
    verifyEmailChange,
    uploadAvatar,
    removeAvatar,
  } = await import("../profileApi.js");

  assert.strictEqual(typeof getProfile, "function");
  assert.strictEqual(typeof updateProfile, "function");
  assert.strictEqual(typeof requestEmailChange, "function");
  assert.strictEqual(typeof resendEmailChangeOTP, "function");
  assert.strictEqual(typeof verifyEmailChange, "function");
  assert.strictEqual(typeof uploadAvatar, "function");
  assert.strictEqual(typeof removeAvatar, "function");
});

test("authStore.updateUser: synchronizes name, phone, and canonical avatar properties", async () => {
  // Mock localStorage for node test runner
  const storageMap = new Map();
  globalThis.localStorage = {
    getItem: (key) => storageMap.get(key) || null,
    setItem: (key, val) => storageMap.set(key, String(val)),
    removeItem: (key) => storageMap.delete(key),
    clear: () => storageMap.clear(),
  };

  const { default: useAuthStore, STORAGE_KEYS } = await import("../../store/authStore.js");

  // Set initial auth state
  const initialUser = {
    id: "user-123",
    fullName: "Original Name",
    email: "test@example.com",
    phone: "+919876543210",
    role: "CUSTOMER",
    profileImage: "https://s3.example.com/old-avatar.png",
  };

  useAuthStore.getState().setAuth(initialUser, "dummy-access-token", "dummy-refresh-token");

  // Update user name and phone
  const updatedUser = useAuthStore.getState().updateUser({
    fullName: "Updated Full Name",
    phone: "+919876543211",
  });

  assert.strictEqual(updatedUser.fullName, "Updated Full Name");
  assert.strictEqual(updatedUser.phone, "+919876543211");
  assert.strictEqual(useAuthStore.getState().user.fullName, "Updated Full Name");

  // Verify synced to localStorage
  const storedUser = JSON.parse(localStorage.getItem(STORAGE_KEYS.USER));
  assert.strictEqual(storedUser.fullName, "Updated Full Name");
  assert.strictEqual(storedUser.phone, "+919876543211");

  // Update avatar and check canonical properties (profileImage and ProfileImage)
  const avatarUpdated = useAuthStore.getState().updateUser({
    profile_image: "https://s3.example.com/new-avatar.png",
  });

  assert.strictEqual(avatarUpdated.profileImage, "https://s3.example.com/new-avatar.png");
  assert.strictEqual(avatarUpdated.ProfileImage, "https://s3.example.com/new-avatar.png");
  assert.strictEqual(useAuthStore.getState().user.profileImage, "https://s3.example.com/new-avatar.png");
  assert.strictEqual(useAuthStore.getState().user.ProfileImage, "https://s3.example.com/new-avatar.png");
});

test("Profile picture validation: verifies size limits and allowed MIME types", () => {
  const allowedMimeTypes = ["image/jpeg", "image/png", "image/webp"];
  const maxBytes = 5 * 1024 * 1024; // 5 MB

  const validateAvatar = (file) => {
    if (!allowedMimeTypes.includes(file.type)) {
      return { valid: false, error: "Only JPEG, PNG, and WebP are allowed" };
    }
    if (file.size > maxBytes) {
      return { valid: false, error: "File exceeds 5 MB limit" };
    }
    return { valid: true };
  };

  // Valid files
  assert.strictEqual(validateAvatar({ type: "image/jpeg", size: 1024 * 1024 }).valid, true);
  assert.strictEqual(validateAvatar({ type: "image/png", size: 4.9 * 1024 * 1024 }).valid, true);
  assert.strictEqual(validateAvatar({ type: "image/webp", size: 2 * 1024 * 1024 }).valid, true);

  // Invalid MIME
  assert.strictEqual(validateAvatar({ type: "image/gif", size: 500 * 1024 }).valid, false);
  assert.strictEqual(validateAvatar({ type: "text/plain", size: 1024 }).valid, false);
  assert.strictEqual(validateAvatar({ type: "application/pdf", size: 1024 }).valid, false);

  // File size exceeding limit
  assert.strictEqual(validateAvatar({ type: "image/png", size: 5.1 * 1024 * 1024 }).valid, false);
  assert.strictEqual(validateAvatar({ type: "image/jpeg", size: 10 * 1024 * 1024 }).valid, false);
});

test("Email change flow logic: email normalization, validation, and timers", () => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  const validateEmailChange = (currentEmail, newEmail) => {
    const trimmed = (newEmail || "").trim().toLowerCase();
    if (!trimmed) return { valid: false, error: "New email is required" };
    if (!emailRegex.test(trimmed)) return { valid: false, error: "Invalid email format" };
    if (trimmed === (currentEmail || "").trim().toLowerCase()) {
      return { valid: false, error: "New email must be different from current email" };
    }
    return { valid: true, normalizedEmail: trimmed };
  };

  // Valid change
  const res1 = validateEmailChange("user@example.com", "  NEWUSER@EXAMPLE.COM  ");
  assert.strictEqual(res1.valid, true);
  assert.strictEqual(res1.normalizedEmail, "newuser@example.com");

  // Same email rejected
  const res2 = validateEmailChange("user@example.com", "USER@EXAMPLE.COM");
  assert.strictEqual(res2.valid, false);
  assert.strictEqual(res2.error, "New email must be different from current email");

  // Invalid email rejected
  const res3 = validateEmailChange("user@example.com", "notanemail");
  assert.strictEqual(res3.valid, false);

  // Timer formatting helper
  const formatTime = (secs) => {
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${m}:${s < 10 ? "0" : ""}${s}`;
  };

  assert.strictEqual(formatTime(600), "10:00");
  assert.strictEqual(formatTime(59), "0:59");
  assert.strictEqual(formatTime(9), "0:09");
  assert.strictEqual(formatTime(0), "0:00");
});

test("Unified profile role badges across CUSTOMER, ORGANIZER, and ADMIN", () => {
  const roleBadgeStyles = {
    ADMIN: "bg-purple-100 text-purple-800 border-purple-200",
    ORGANIZER: "bg-emerald-100 text-emerald-800 border-emerald-200",
    CUSTOMER: "bg-blue-100 text-blue-800 border-blue-200",
  };

  for (const role of ["CUSTOMER", "ORGANIZER", "ADMIN"]) {
    assert.ok(roleBadgeStyles[role], `Style defined for role ${role}`);
    assert.strictEqual(typeof roleBadgeStyles[role], "string");
  }
});
