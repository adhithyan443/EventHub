import { create } from "zustand";
import { applyAsOrganizer } from "../api/organizerApi";

const DRAFT_STORAGE_KEY = "eventhub_organizer_draft";

const initialData = {
  currentStep: 1,

  // Non-sensitive information (can be saved to draft)
  businessName: "",
  businessType: "",
  description: "",
  phone: "",
  website: "",

  // Business Address fields
  addressLine: "",
  city: "",
  state: "",
  country: "",
  postalCode: "",

  // Sensitive information (kept in memory only, NEVER stored in localStorage)
  panNumber: "",
  gstNumber: "",
  bankName: "",
  accountHolderName: "",
  accountNumber: "",
  confirmAccount: "",
  ifscCode: "",

  verificationDocument: null,
  verificationDocumentUrl: "",

  businessLogo: null,
  businessLogoPreview: "",
  businessLogoUrl: "",

  // Status & Submission
  termsAccepted: false,
  applicationSubmitted: false,
  applicationStatus: null,
  applicationId: null,
  submissionDate: null,

  // Submission state
  isSubmitting: false,
  submitError: null,
};

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
      addressLine: parsed.addressLine || "",
      city: parsed.city || "",
      state: parsed.state || "",
      country: parsed.country || "",
      postalCode: parsed.postalCode || "",
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
    set({
      businessLogo: file,
      businessLogoPreview: previewUrl,
    });
  },

  removeBusinessLogo: () => {
    set({
      businessLogo: null,
      businessLogoPreview: "",
    });
  },

  saveDraft: () => {
    const state = get();

    const draft = {
      businessName: state.businessName,
      businessType: state.businessType,
      description: state.description,
      phone: state.phone,
      website: state.website,
      addressLine: state.addressLine,
      city: state.city,
      state: state.state,
      country: state.country,
      postalCode: state.postalCode,
      currentStep: state.currentStep,
    };

    try {
      localStorage.setItem(
        DRAFT_STORAGE_KEY,
        JSON.stringify(draft)
      );

      return true;
    } catch {
      return false;
    }
  },

  submitApplication: async () => {
    const state = get();

    set({
      isSubmitting: true,
      submitError: null,
    });

    try {
      // Build the payload using the exact backend snake_case contract
      const applicationData = {
        business_name: state.businessName?.trim(),
        business_type: state.businessType,
        business_description: state.description?.trim(),
        contact_phone: state.phone?.trim(),
        website: state.website?.trim() || "",

        address_line: state.addressLine?.trim(),
        city: state.city?.trim(),
        state: state.state?.trim(),
        country: state.country?.trim(),
        postal_code: state.postalCode?.trim(),

        bank_name: state.bankName?.trim(),
        account_holder_name: state.accountHolderName?.trim(),
        account_number: state.accountNumber?.trim(),
        ifsc_code: state.ifscCode?.trim().toUpperCase(),

        gst_number: state.gstNumber?.trim().toUpperCase() || "",
        pan_number: state.panNumber?.trim().toUpperCase(),

        logo_url: state.businessLogoUrl || "",
        verification_document_url: state.verificationDocumentUrl || "",
      };

      // Submit the application to the real backend
      const response = await applyAsOrganizer(applicationData);

      // Read the application information returned by the backend
      const application = response?.data || response;

      set({
        applicationSubmitted: true,
        applicationStatus: application?.status || "PENDING",
        applicationId:
          application?.id ||
          application?.applicationId ||
          null,
        submissionDate:
          application?.createdAt || new Date().toISOString(),
        isSubmitting: false,
        submitError: null,
      });

      // Remove the saved non-sensitive draft after successful submission
      try {
        localStorage.removeItem(DRAFT_STORAGE_KEY);
      } catch {
        // Ignore localStorage cleanup failures.
      }

      return response;
    } catch (error) {
      // Keep the error safe; never log sensitive request data
      const message =
        error?.response?.data?.message ||
        error?.message ||
        "Failed to submit organizer application";

      set({
        isSubmitting: false,
        submitError: message,
      });

      throw error;
    }
  },

  resetForm: () => {
    try {
      localStorage.removeItem(DRAFT_STORAGE_KEY);
    } catch {
      // Ignore localStorage cleanup failures.
    }

    set({ ...initialData });
  },
}));

export default useOrganizerStore;