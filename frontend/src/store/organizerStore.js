import { create } from "zustand";

const DRAFT_STORAGE_KEY = "eventhub_organizer_draft";

const initialData = {
  currentStep: 1,

  // Non-sensitive information (can be saved to draft)
  businessName: "",
  businessType: "",
  description: "",
  phone: "",
  website: "",

  // Sensitive information (kept in memory only, NEVER stored in localStorage)
  panNumber: "",
  gstNumber: "",
  bankName: "",
  accountHolder: "",
  accountNumber: "",
  confirmAccount: "",
  ifscCode: "",
  verificationDocument: null, // { name, size, type, file }
  businessLogo: null, // File
  businessLogoPreview: "", // Object URL or data URL

  // Status & Submission
  termsAccepted: false,
  applicationSubmitted: false,
  applicationStatus: null, // 'PENDING'
  applicationId: null,
  submissionDate: null,
};

// Check if a draft exists in localStorage
function getSavedDraft() {
  try {
    const raw = localStorage.getItem(DRAFT_STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    return {
      businessName: parsed.businessName || "",
      businessType: parsed.businessType || "",
      description: parsed.description || "",
      phone: parsed.phone || "",
      website: parsed.website || "",
      currentStep: parsed.currentStep || 1,
    };
  } catch {
    return null;
  }
}

const savedDraft = getSavedDraft();

const useOrganizerStore = create((set, get) => ({
  ...initialData,
  ...(savedDraft || {}),

  setField: (field, value) => {
    set({ [field]: value });
  },

  setMultipleFields: (fields) => {
    set(fields);
  },

  setCurrentStep: (step) => {
    set({ currentStep: step });
  },

  setVerificationDocument: (doc) => {
    set({ verificationDocument: doc });
  },

  removeVerificationDocument: () => {
    set({ verificationDocument: null });
  },

  setBusinessLogo: (file, previewUrl) => {
    set({ businessLogo: file, businessLogoPreview: previewUrl });
  },

  removeBusinessLogo: () => {
    set({ businessLogo: null, businessLogoPreview: "" });
  },

  saveDraft: () => {
    const state = get();
    const draft = {
      businessName: state.businessName,
      businessType: state.businessType,
      description: state.description,
      phone: state.phone,
      website: state.website,
      currentStep: state.currentStep,
    };
    try {
      localStorage.setItem(DRAFT_STORAGE_KEY, JSON.stringify(draft));
      return true;
    } catch {
      return false;
    }
  },

  submitApplication: () => {
    const now = new Date();
    const formattedDate = now.toLocaleDateString("en-GB", {
      day: "numeric",
      month: "long",
      year: "numeric",
    });

    // Generate random application ID or default APP001
    const randNum = Math.floor(100 + Math.random() * 900);
    const appId = `APP${randNum}`;

    set({
      applicationSubmitted: true,
      applicationStatus: "PENDING",
      applicationId: appId,
      submissionDate: formattedDate,
    });

    // Clear saved non-sensitive draft on submission
    try {
      localStorage.removeItem(DRAFT_STORAGE_KEY);
    } catch {
      // ignore
    }

    return appId;
  },

  resetForm: () => {
    try {
      localStorage.removeItem(DRAFT_STORAGE_KEY);
    } catch {
      // ignore
    }
    set({ ...initialData });
  },
}));

export default useOrganizerStore;
