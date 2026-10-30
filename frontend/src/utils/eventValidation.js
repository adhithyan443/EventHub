import {
  TICKET_MODES,
  isOnlineEvent,
  isPhysicalEvent,
  isHybridEvent,
  parseAgeRestriction,
} from "../constants/eventConstants.js";

export const isUUID = (str) =>
  typeof str === "string" &&
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
    str
  );

export const VALID_VISIBILITIES = ["PUBLIC", "PRIVATE"];
export const VALID_REFUND_POLICIES = ["FULL", "PARTIAL", "NO_REFUND"];
export const VALID_CANCELLATION_DEADLINES = [
  "24 hours before event",
  "48 hours before event",
  "7 days before event",
  "14 days before event",
];

export const isValidEmail = (email) =>
  typeof email === "string" && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());

export const isValidPhone = (phone) =>
  typeof phone === "string" && /^\+?[0-9\s().-]{7,20}$/.test(phone.trim());

export const parseDeadlineHours = (deadlineStr) => {
  if (typeof deadlineStr === "number" && deadlineStr > 0) return deadlineStr;
  const str = String(deadlineStr || "").trim();
  if (str === "24 hours before event" || str === "24") return 24;
  if (str === "48 hours before event" || str === "48") return 48;
  if (str === "7 days before event" || str === "168") return 168;
  if (str === "14 days before event" || str === "336") return 336;
  const matchHours = str.match(/^(\d+)\s*hours?/i);
  if (matchHours) return Number(matchHours[1]);
  const matchDays = str.match(/^(\d+)\s*days?/i);
  if (matchDays) return Number(matchDays[1]) * 24;
  const parsed = parseInt(str, 10);
  return !Number.isNaN(parsed) && parsed > 0 ? parsed : 48;
};

export const extractEventId = (res) =>
  res?.data?.data?.event?.ID ||
  res?.data?.data?.event?.id ||
  res?.data?.event?.ID ||
  res?.data?.event?.id ||
  res?.data?.data?.ID ||
  res?.data?.data?.id ||
  res?.data?.ID ||
  res?.data?.id ||
  res?.event?.ID ||
  res?.event?.id ||
  res?.ID ||
  res?.id;

export const formatSubmissionErrorMessage = (err, fallbackMsg) => {
  const backendMsg = err?.response?.data?.message || err?.response?.data?.error;
  if (backendMsg === "invalid event data") {
    return "Event submission failed: Some event fields are invalid or incomplete. Please check your event details, venue address, and contact information.";
  }
  return backendMsg || err?.message || fallbackMsg;
};

/**
 * Validates minimum requirements genuinely necessary to create or update an event draft in the database.
 * Does NOT enforce publishing-readiness checks like organizer cancellation policy acceptance or future event date.
 */
export const validateEventForDraft = ({
  basicInformation,
  locationType,
  onlineUrl,
  venue,
  dateTime,
  eventDetails,
  ticketMode,
  ticketTypes,
  seatingConfig,
  ticketSalesSettings,
  isEditMode = false,
}) => {
  const isOnline = isOnlineEvent(locationType);
  const isPhysical = isPhysicalEvent(locationType);
  const isHybrid = isHybridEvent(locationType);

  // 1. Basic Information
  if (!basicInformation?.title?.trim()) {
    return "Event title is required.";
  }
  if (!basicInformation?.categoryId) {
    return "Please select an event category.";
  }
  if (!basicInformation?.description?.trim()) {
    return "Event description is required.";
  }
  if (!isEditMode && !basicInformation?.bannerFile && !basicInformation?.banner) {
    return "Please upload an event banner.";
  }
  if (!basicInformation?.language?.trim()) {
    return "Please specify the event language.";
  }
  const ageRestriction = parseAgeRestriction(basicInformation?.ageRestriction);
  if (!Number.isInteger(ageRestriction) || ageRestriction < 0 || ageRestriction > 99) {
    return "Age restriction must be a valid number between 0 and 99.";
  }

  // 2. Schedule
  if (!dateTime?.eventDate) {
    return "Event date is required.";
  }
  if (!dateTime.isAllDay) {
    if (!dateTime.startTime || !dateTime.endTime) {
      return "Start time and end time are required.";
    }
    if (dateTime.startTime >= dateTime.endTime) {
      return "End time must be later than the start time.";
    }
  }

  // 3. Location
  if (isOnline || isHybrid) {
    if (!onlineUrl?.trim()) {
      return "Online events require a virtual stream or meeting URL.";
    }
    try {
      const parsed = new URL(onlineUrl.trim());
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
        return "Online event URL must use http:// or https://.";
      }
    } catch {
      return "Please provide a valid online event URL.";
    }
  }
  if (isPhysical || isHybrid) {
    if (!venue?.name?.trim()) {
      return "Please select a physical venue.";
    }
    if (!venue.address?.trim()) {
      return "Venue address is required.";
    }
    if (!venue.city?.trim()) {
      return "Venue city is required.";
    }
    if (!venue.state?.trim()) {
      return "Venue state is required. Please re-select the venue from Google Places to capture its state.";
    }
    if (!venue.country?.trim()) {
      return "Venue country is required.";
    }
    if (!venue.google_place_id?.trim()) {
      return "Google Place ID is required. Please select an established venue using Google Places.";
    }
    if (
      !Number.isFinite(Number(venue.latitude)) ||
      !Number.isFinite(Number(venue.longitude)) ||
      Number(venue.latitude) < -90 ||
      Number(venue.latitude) > 90 ||
      Number(venue.longitude) < -180 ||
      Number(venue.longitude) > 180
    ) {
      return "Valid venue coordinates are required.";
    }
  }

  // 4. Ticket Sales Settings
  if (!ticketSalesSettings?.salesStartDate || !ticketSalesSettings?.salesEndDate) {
    return "Ticket sales start date and end date are required.";
  }
  if (ticketSalesSettings.salesStartDate > ticketSalesSettings.salesEndDate) {
    return "Ticket sales start date must be before or equal to the end date.";
  }
  if (dateTime.eventDate && ticketSalesSettings.salesEndDate > dateTime.eventDate) {
    return "Ticket sales end date cannot be after the event date.";
  }
  const maxTickets = Number(ticketSalesSettings.maxTicketsPerBooking);
  if (!Number.isInteger(maxTickets) || maxTickets <= 0) {
    return "Maximum tickets per booking must be a positive number.";
  }

  // 5. Ticket Configuration
  if (!isOnline && ticketMode === TICKET_MODES.SEATED) {
    const categories = seatingConfig?.categories || [];
    if (categories.length === 0) {
      return "Reserved seating requires at least one seating category.";
    }
    for (const cat of categories) {
      const price = Number(String(cat.price ?? "").replace(/[₹,\s]/g, ""));
      if (Number.isNaN(price) || price < 0) {
        return `Category "${cat.name}" has an invalid price.`;
      }
      if (!cat.rows || cat.rows.length === 0) {
        return `Category "${cat.name}" must contain at least one row.`;
      }
    }
  } else {
    if (!ticketTypes || ticketTypes.length === 0) {
      return "Please create at least one ticket tier.";
    }
    for (const ticket of ticketTypes) {
      if (!ticket.name?.trim()) {
        return "All ticket tiers must have a name.";
      }
      const price = Number(ticket.price);
      if (Number.isNaN(price) || price < 0) {
        return `Ticket tier "${ticket.name}" must have a non-negative price.`;
      }
      const capacity = Number(ticket.capacity);
      if (!Number.isInteger(capacity) || capacity <= 0) {
        return `Ticket tier "${ticket.name}" capacity must be a positive number.`;
      }
    }
  }

  // 6. Cancellation Policy
  const cancellationPolicy = eventDetails?.cancellationPolicy;
  if (!cancellationPolicy) {
    return "Cancellation policy is required.";
  }
  if (cancellationPolicy.allowCancellation) {
    if (!VALID_CANCELLATION_DEADLINES.includes(cancellationPolicy.cancellationDeadline)) {
      return "Please select a valid cancellation deadline.";
    }
    if (!VALID_REFUND_POLICIES.includes(cancellationPolicy.refundPolicy)) {
      return "Please select a valid refund policy.";
    }
    const refundPct = Number(cancellationPolicy.refundPercentage);
    if (cancellationPolicy.refundPolicy === "FULL" && refundPct !== 100) {
      return "Full refund policy requires 100% refund percentage.";
    }
    if (
      cancellationPolicy.refundPolicy === "PARTIAL" &&
      (Number.isNaN(refundPct) || refundPct <= 0 || refundPct >= 100)
    ) {
      return "Partial refund policy requires a refund percentage between 1% and 99%.";
    }
    if (cancellationPolicy.refundPolicy === "NO_REFUND" && refundPct !== 0) {
      return "No refund policy requires 0% refund percentage.";
    }
  }

  // 7. Contact Information (Optional, but must be complete if any field is provided)
  const contact = eventDetails?.contactInformation || {};
  const contactName = contact.name?.trim() || "";
  const contactPhone = contact.phone?.trim() || "";
  const contactEmail = contact.email?.trim() || "";
  const hasAnyContact = contactName !== "" || contactPhone !== "" || contactEmail !== "";

  if (hasAnyContact) {
    if (!contactName) {
      return "Contact name is required when contact details are provided.";
    }
    if (contactName.length > 100) {
      return "Contact name cannot exceed 100 characters.";
    }
    if (!contactPhone) {
      return "Contact phone number is required when contact details are provided.";
    }
    if (contactPhone.length > 20 || !isValidPhone(contactPhone)) {
      return "Please provide a valid organizer contact phone number.";
    }
    if (!contactEmail) {
      return "Contact email is required when contact details are provided.";
    }
    if (contactEmail.length > 255 || !isValidEmail(contactEmail)) {
      return "Please provide a valid organizer contact email address.";
    }
  }

  return null;
};

/**
 * Validates publishing-readiness requirements in addition to draft validity.
 */
export const validateEventForPublish = (args) => {
  const draftError = validateEventForDraft(args);
  if (draftError) {
    return draftError;
  }

  const { dateTime, eventDetails } = args;

  // 1. Publishing readiness: Event date must not be in the past
  if (dateTime?.eventDate) {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const eventDateObj = new Date(dateTime.eventDate + "T00:00:00");
    if (eventDateObj < today) {
      return "Event date cannot be in the past.";
    }
  }

  // 2. Publishing readiness: Acceptance of organizer cancellation policy
  const cancellationPolicy = eventDetails?.cancellationPolicy;
  if (cancellationPolicy?.organizerPolicyAccepted !== true) {
    return "You must accept the organizer cancellation policy before publishing.";
  }

  // 3. Publishing readiness: Visibility
  if (!VALID_VISIBILITIES.includes(eventDetails?.visibility)) {
    return "Please select a valid event visibility.";
  }

  return null;
};
