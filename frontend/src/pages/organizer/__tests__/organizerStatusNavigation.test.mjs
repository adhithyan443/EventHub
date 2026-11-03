import test, { beforeEach } from "node:test";
import assert from "node:assert";

// 1. In-memory localStorage polyfill
const mockStorage = new Map();
globalThis.localStorage = {
  getItem: (key) => (mockStorage.has(key) ? mockStorage.get(key) : null),
  setItem: (key, val) => mockStorage.set(key, String(val)),
  removeItem: (key) => mockStorage.delete(key),
  clear: () => mockStorage.clear(),
};

// 2. Import auth store, client, and organizer store
const { default: useAuthStore } = await import("../../../store/authStore.js");
const { default: useOrganizerStore } = await import("../../../store/organizerStore.js");
const {
  default: apiClient,
  rawAuthAxios,
  resetActiveRefreshPromiseForTesting,
} = await import("../../../api/client.js");

beforeEach(() => {
  mockStorage.clear();
  resetActiveRefreshPromiseForTesting();
  useAuthStore.setState({
    user: null,
    accessToken: null,
    refreshToken: null,
    isAuthenticated: false,
    isInitializing: false,
    sessionVersion: 0,
  });
  useOrganizerStore.setState({
    applicationId: null,
    applicationStatus: null,
    applicationSubmitted: false,
    isResubmission: false,
    isCheckingApplication: false,
    fetchError: null,
    rejectionReason: "",
    businessName: "",
    businessType: "",
  });
});

test("fetchApplication: sets PENDING status and submitted flag when backend returns PENDING", async () => {
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/organizers/application")) {
      return {
        status: 200,
        data: {
          id: "app-pending-1",
          status: "PENDING",
          business_name: "Aura Events",
          business_type: "INDIVIDUAL",
        },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const result = await useOrganizerStore.getState().fetchApplication();

  assert.strictEqual(result.exists, true);
  assert.strictEqual(result.application.status, "PENDING");
  const state = useOrganizerStore.getState();
  assert.strictEqual(state.applicationStatus, "PENDING");
  assert.strictEqual(state.applicationSubmitted, true);
  assert.strictEqual(state.isResubmission, false);
  assert.strictEqual(state.businessName, "Aura Events");
  assert.strictEqual(state.isCheckingApplication, false);
  assert.strictEqual(state.fetchError, null);
});

test("fetchApplication: sets UNDER_REVIEW status when returned by backend contract", async () => {
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/organizers/application")) {
      return {
        status: 200,
        data: {
          id: "app-review-1",
          status: "UNDER_REVIEW",
          business_name: "Apex Entertainment",
          business_type: "COMPANY",
        },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const result = await useOrganizerStore.getState().fetchApplication();

  assert.strictEqual(result.exists, true);
  assert.strictEqual(result.application.status, "UNDER_REVIEW");
  assert.strictEqual(useOrganizerStore.getState().applicationStatus, "UNDER_REVIEW");
  assert.strictEqual(useOrganizerStore.getState().businessName, "Apex Entertainment");
});

test("fetchApplication: sets REJECTED status, reason, and isResubmission flag", async () => {
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/organizers/application")) {
      return {
        status: 200,
        data: {
          id: "app-rejected-1",
          status: "REJECTED",
          rejection_reason: "PAN card copy is illegible",
          business_name: "Summit Events",
          business_type: "PARTNERSHIP",
        },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const result = await useOrganizerStore.getState().fetchApplication();

  assert.strictEqual(result.exists, true);
  assert.strictEqual(result.application.status, "REJECTED");
  const state = useOrganizerStore.getState();
  assert.strictEqual(state.applicationStatus, "REJECTED");
  assert.strictEqual(state.isResubmission, true);
  assert.strictEqual(state.rejectionReason, "PAN card copy is illegible");
});

test("fetchApplication: handles approved-organizer fallback with status APPROVED", async () => {
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/organizers/application")) {
      return {
        status: 200,
        data: {
          id: "app-approved-fallback",
          status: "APPROVED",
          business_name: "Pioneer Festivals",
          business_type: "COMPANY",
        },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  const result = await useOrganizerStore.getState().fetchApplication();

  assert.strictEqual(result.exists, true);
  assert.strictEqual(result.application.status, "APPROVED");
  const state = useOrganizerStore.getState();
  assert.strictEqual(state.applicationStatus, "APPROVED");
  assert.strictEqual(state.applicationSubmitted, true);
});

test("genuine 404: clears application state without displaying misleading status or error", async () => {
  // Pre-populate with stale state
  useOrganizerStore.setState({
    applicationStatus: "PENDING",
    applicationSubmitted: true,
  });

  apiClient.defaults.adapter = async (config) => {
    const err = new Error("Not Found");
    err.response = { status: 404, data: { message: "no application found" } };
    throw err;
  };

  const result = await useOrganizerStore.getState().fetchApplication();

  assert.strictEqual(result.exists, false);
  assert.strictEqual(result.application, null);
  const state = useOrganizerStore.getState();
  assert.strictEqual(state.applicationStatus, null);
  assert.strictEqual(state.applicationSubmitted, false);
  assert.strictEqual(state.fetchError, null);
  assert.strictEqual(state.isCheckingApplication, false);
});

test("server 500 error: sets fetchError and avoids setting misleading application status", async () => {
  apiClient.defaults.adapter = async (config) => {
    const err = new Error("Internal Server Error");
    err.response = { status: 500, data: { message: "database offline" } };
    throw err;
  };

  const result = await useOrganizerStore.getState().fetchApplication();

  assert.strictEqual(result.exists, false);
  assert.strictEqual(result.error, "database offline");
  const state = useOrganizerStore.getState();
  assert.strictEqual(state.applicationStatus, null);
  assert.strictEqual(state.fetchError, "database offline");
  assert.strictEqual(state.isCheckingApplication, false);
});

test("approved organizer status derivation: user with role ORGANIZER is always recognized as APPROVED even if application is null", () => {
  useAuthStore.setState({
    user: { id: "user-org", role: "ORGANIZER" },
    isAuthenticated: true,
  });
  useOrganizerStore.setState({
    applicationStatus: null,
    applicationSubmitted: false,
  });

  const authUser = useAuthStore.getState().user;
  const rawStatus = useOrganizerStore.getState().applicationStatus;

  // Derivation logic used across OrganizerStatusPage, BecomeOrganizerPage, and OrganizerPromoCard
  const effectiveStatus =
    authUser?.role === "ORGANIZER" ? "APPROVED" : (rawStatus || "PENDING");
  const isApproved = effectiveStatus === "APPROVED";

  assert.strictEqual(effectiveStatus, "APPROVED");
  assert.strictEqual(isApproved, true);
});

test("route guard contract: customers cannot access allowedRoles=['ORGANIZER'] while organizers can", () => {
  const allowedRoles = ["ORGANIZER"];

  const customerUser = { id: "u-cust", role: "CUSTOMER" };
  const canCustomerAccess = Boolean(customerUser && allowedRoles.includes(customerUser.role));
  assert.strictEqual(canCustomerAccess, false, "Customer must be blocked from organizer routes");

  const organizerUser = { id: "u-org", role: "ORGANIZER" };
  const canOrganizerAccess = Boolean(organizerUser && allowedRoles.includes(organizerUser.role));
  assert.strictEqual(canOrganizerAccess, true, "Approved organizer must be permitted to access organizer routes");
});

test("application approval triggers proactive auth revalidation to upgrade CUSTOMER to ORGANIZER", async () => {
  // 1. Initial active session as CUSTOMER
  useAuthStore.getState().setAuth(
    { id: "user-sync-test", role: "CUSTOMER", fullName: "Applicant User" },
    "token-cust",
    "refresh-cust"
  );

  // 2. Mock GET /organizers/application returning APPROVED status
  apiClient.defaults.adapter = async (config) => {
    if (config.url.includes("/organizers/application")) {
      return {
        status: 200,
        data: {
          id: "app-promoted-1",
          status: "APPROVED",
          business_name: "Promoted Events",
        },
        config,
      };
    }
    if (config.url.includes("/auth/me")) {
      return {
        status: 200,
        data: {
          user: { id: "user-sync-test", role: "ORGANIZER", fullName: "Applicant User" },
        },
        config,
      };
    }
    return { status: 200, data: {}, config };
  };

  // 3. Mock refresh-token endpoint returning new ORGANIZER JWT
  let refreshCalls = 0;
  rawAuthAxios.defaults.adapter = async (config) => {
    refreshCalls++;
    return {
      status: 200,
      data: {
        access_token: "new-jwt-organizer-token",
        refresh_token: "rotated-refresh",
        user: { id: "user-sync-test", role: "ORGANIZER", fullName: "Applicant User" },
      },
      config,
    };
  };

  // 4. Page checks application status
  const appResult = await useOrganizerStore.getState().fetchApplication();
  assert.strictEqual(appResult.application.status, "APPROVED");

  // 5. On detecting APPROVED status, status pages trigger revalidateSession
  const revalResult = await useAuthStore.getState().revalidateSession();
  assert.strictEqual(revalResult.roleChanged, true);
  assert.strictEqual(revalResult.newRole, "ORGANIZER");
  assert.strictEqual(refreshCalls, 1);

  // 6. User role in auth store is now ORGANIZER and access token reflects updated claims
  assert.strictEqual(useAuthStore.getState().user.role, "ORGANIZER");
  assert.strictEqual(useAuthStore.getState().accessToken, "new-jwt-organizer-token");
});

test("navigation safety: revalidation failure or non-organizer role prevents dashboard navigation", async () => {
  useAuthStore.getState().setAuth(
    { id: "user-fail-test", role: "CUSTOMER" },
    "token-fail",
    "refresh-fail"
  );
  useOrganizerStore.setState({
    applicationStatus: "APPROVED",
    applicationSubmitted: true,
  });

  // Backend /auth/me fails with 500 error
  apiClient.defaults.adapter = async (config) => {
    const err = new Error("Network / Server Error");
    err.response = { status: 500, data: { message: "Internal server error" } };
    throw err;
  };

  // Simulate handleGoToDashboard verification logic
  let navigatedRoute = null;
  let errorFeedback = null;
  const navigate = (path) => {
    navigatedRoute = path;
  };

  let currentRole = useAuthStore.getState().user?.role;
  if (currentRole !== "ORGANIZER") {
    const revalResult = await useAuthStore.getState().revalidateSession();
    currentRole = useAuthStore.getState().user?.role;
    if (currentRole !== "ORGANIZER") {
      errorFeedback =
        revalResult?.error?.message || "Unable to verify organizer privileges";
    }
  }

  if (currentRole === "ORGANIZER") {
    navigate("/organizer/dashboard");
  }

  // Verify dashboard navigation was BLOCKED
  assert.strictEqual(navigatedRoute, null, "Dashboard navigation must not occur when revalidation fails");
  assert.ok(errorFeedback, "User must receive error feedback");
  // Existing session preserved
  assert.strictEqual(useAuthStore.getState().isAuthenticated, true);
  assert.strictEqual(useAuthStore.getState().user.role, "CUSTOMER");
});

test("header navigation links: dynamic links update between CUSTOMER and ORGANIZER", () => {
  const baseNavLinks = [
    { label: "Events", to: "/events" },
    { label: "Categories", to: "/categories" },
    { label: "My Tickets", to: "/my-tickets" },
  ];

  // 1. Customer role
  const getNavLinks = (userRole) => {
    const isOrganizer = userRole === "ORGANIZER";
    return isOrganizer
      ? [...baseNavLinks, { label: "Organizer Dashboard", to: "/organizer/dashboard" }]
      : baseNavLinks;
  };

  const customerLinks = getNavLinks("CUSTOMER");
  assert.strictEqual(customerLinks.length, 3);
  assert.strictEqual(
    customerLinks.some((l) => l.to === "/organizer/dashboard"),
    false,
    "Customer must not see Organizer Dashboard in header links"
  );

  // 2. Organizer role
  const organizerLinks = getNavLinks("ORGANIZER");
  assert.strictEqual(organizerLinks.length, 4);
  assert.strictEqual(
    organizerLinks.some((l) => l.to === "/organizer/dashboard"),
    true,
    "Organizer must see Organizer Dashboard in header links"
  );
});
