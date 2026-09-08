/**
 * Constants for the EventHub Organizer and Event Creation flow.
 */

export const TICKET_MODES = Object.freeze({
  GENERAL: "GENERAL",
  SEATED: "SEATED",
});

export const LOCATION_TYPES = Object.freeze({
  PHYSICAL: "PHYSICAL",
  ONLINE: "ONLINE",
  HYBRID: "HYBRID",
});

export const ORGANIZER_ROUTES = Object.freeze({
  DASHBOARD: "/organizer/dashboard",
  CREATE_STEP_1: "/organizer/events/create/step-1",
  CREATE_STEP_2: "/organizer/events/create/step-2",
  CREATE_STEP_3: "/organizer/events/create/step-3",
  CREATE_TICKET_TYPES: "/organizer/events/create/ticket-types",
  CREATE_SEAT_CONFIG: "/organizer/events/create/seat-configuration",
  CREATE_REVIEW: "/organizer/events/create/review",
  CREATE_EVENT_ALIAS: "/organizer/create-event",
  MY_EVENTS: "/organizer/events",
  BOOKINGS: "/organizer/bookings",
  ATTENDEES: "/organizer/attendees",
  REPORTS: "/organizer/reports",
  SETTINGS: "/organizer/settings",
});

// Alias EVENT_TYPES to LOCATION_TYPES
export const EVENT_TYPES = LOCATION_TYPES;

// Helper functions for event-type logic
export const isPhysicalEvent = (type) => type === LOCATION_TYPES.PHYSICAL;
export const isOnlineEvent = (type) => type === LOCATION_TYPES.ONLINE;
export const isHybridEvent = (type) => type === LOCATION_TYPES.HYBRID;

export const hasPhysicalComponent = (type) =>
  type === LOCATION_TYPES.PHYSICAL || type === LOCATION_TYPES.HYBRID;

export const hasOnlineComponent = (type) =>
  type === LOCATION_TYPES.ONLINE || type === LOCATION_TYPES.HYBRID;

export const canUseReservedSeating = (type) =>
  type === LOCATION_TYPES.PHYSICAL || type === LOCATION_TYPES.HYBRID;

export const canUseVenue = (type) =>
  type === LOCATION_TYPES.PHYSICAL || type === LOCATION_TYPES.HYBRID;

/**
 * Dynamically computes the steps in the Event Creation Stepper
 * based on event type (locationType) and ticketing mode.
 */
export const getEventCreationSteps = (locationType, ticketMode) => {
  const isOnline = isOnlineEvent(locationType);
  const isSeated = !isOnline && ticketMode === TICKET_MODES.SEATED;

  return [
    {
      step: 1,
      label: "Basic Information",
      path: ORGANIZER_ROUTES.CREATE_STEP_1,
    },
    {
      step: 2,
      label: "Event Details",
      path: ORGANIZER_ROUTES.CREATE_STEP_2,
    },
    {
      step: 3,
      label: isOnline ? "Ticketing" : "Ticket Selection",
      path: ORGANIZER_ROUTES.CREATE_STEP_3,
    },
    {
      step: 4,
      label: isSeated ? "Seat Configuration" : "Ticket Types",
      path: isSeated
        ? ORGANIZER_ROUTES.CREATE_SEAT_CONFIG
        : ORGANIZER_ROUTES.CREATE_TICKET_TYPES,
    },
    {
      step: 5,
      label: "Review & Publish",
      path: ORGANIZER_ROUTES.CREATE_REVIEW,
    },
  ];
};

