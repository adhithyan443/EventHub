/**
 * Utility functions for customer booking cancellation policy evaluation,
 * deadline calculations, and refund estimations.
 */

/**
 * Extracts and parses the start Date from an event schedule object.
 * @param {object} schedule - Event schedule object from backend
 * @returns {Date|null}
 */
export function getEventStartDateTime(schedule) {
  if (!schedule) return null;
  const rawDate = schedule.event_date || schedule.EventDate;
  if (!rawDate) return null;

  const dateObj = new Date(rawDate);
  if (Number.isNaN(dateObj.getTime())) return null;

  const rawTime = schedule.start_time || schedule.StartTime;
  if (typeof rawTime === "string") {
    const match = rawTime.match(/(\d{1,2}):(\d{2})(?::(\d{2}))?/);
    if (match) {
      dateObj.setHours(
        Number(match[1]),
        Number(match[2]),
        match[3] ? Number(match[3]) : 0,
        0
      );
    }
  }
  return dateObj;
}

/**
 * Calculates the exact cancellation deadline Date by subtracting deadline hours
 * from the event start date/time.
 * @param {object} schedule - Event schedule object
 * @param {number} deadlineHours - Hours before event start
 * @returns {Date|null}
 */
export function calculateCancellationDeadline(schedule, deadlineHours = 0) {
  const eventStart = getEventStartDateTime(schedule);
  if (!eventStart) return null;
  const hours = Number(deadlineHours) || 0;
  return new Date(eventStart.getTime() - hours * 60 * 60 * 1000);
}

/**
 * Calculates the expected refund amount and policy details based on the booking snapshot.
 * @param {object} booking - Booking domain object
 * @returns {{ expectedRefund: number, refundPolicy: string, refundPercentage: number }}
 */
export function calculateExpectedRefund(booking) {
  if (!booking) {
    return { expectedRefund: 0, refundPolicy: "NO_REFUND", refundPercentage: 0 };
  }

  const total = Number(booking.Total ?? booking.total ?? 0);
  const rawPolicy = (
    booking.RefundPolicy ??
    booking.refund_policy ??
    "NO_REFUND"
  )
    .toString()
    .toUpperCase()
    .trim();

  const refundPercentage = Number(
    booking.RefundPercentage ??
    booking.refund_percentage ??
    0
  );

  if (total <= 0) {
    return { expectedRefund: 0, refundPolicy: rawPolicy, refundPercentage };
  }

  if (rawPolicy === "FULL") {
    return { expectedRefund: total, refundPolicy: "FULL", refundPercentage: 100 };
  }

  if (rawPolicy === "PARTIAL") {
    if (refundPercentage <= 0) {
      return { expectedRefund: 0, refundPolicy: "PARTIAL", refundPercentage: 0 };
    }
    const amount = Number(((total * refundPercentage) / 100).toFixed(2));
    return { expectedRefund: amount, refundPolicy: "PARTIAL", refundPercentage };
  }

  return { expectedRefund: 0, refundPolicy: "NO_REFUND", refundPercentage: 0 };
}

/**
 * Determines whether a booking is eligible for cancellation by the customer.
 * Evaluates booking status, cancellation allowed flag, deadline vs event time, and ticket status.
 *
 * @param {object} booking - Booking object
 * @param {object} [eventSchedule] - Associated event schedule
 * @param {Array} [tickets] - Optional list of tickets for this booking
 * @returns {{
 *   eligible: boolean,
 *   reason: string|null,
 *   deadlineDate: Date|null,
 *   deadlinePassed: boolean,
 *   deadlineHours: number,
 *   expectedRefund: number,
 *   refundPolicy: string,
 *   refundPercentage: number
 * }}
 */
export function checkCancellationEligibility(booking, eventSchedule = null, tickets = []) {
  if (!booking) {
    return {
      eligible: false,
      reason: "No booking data available.",
      deadlineDate: null,
      deadlinePassed: false,
      deadlineHours: 0,
      expectedRefund: 0,
      refundPolicy: "NO_REFUND",
      refundPercentage: 0,
    };
  }

  const status = (booking.Status ?? booking.status ?? "PENDING").toString().toUpperCase();
  const refundInfo = calculateExpectedRefund(booking);

  const deadlineHours = Number(
    booking.CancellationDeadlineHours ??
    booking.cancellation_deadline_hours ??
    0
  );

  let deadlineDate = null;
  let deadlinePassed = false;

  if (eventSchedule) {
    deadlineDate = calculateCancellationDeadline(eventSchedule, deadlineHours);
    if (deadlineDate && Date.now() > deadlineDate.getTime()) {
      deadlinePassed = true;
    }
  }

  if (status === "CANCELLED") {
    return {
      eligible: false,
      reason: "Booking has already been cancelled.",
      deadlineDate,
      deadlinePassed,
      deadlineHours,
      ...refundInfo,
    };
  }

  if (status !== "CONFIRMED") {
    return {
      eligible: false,
      reason: `Booking cannot be cancelled in its current state (${status}).`,
      deadlineDate,
      deadlinePassed,
      deadlineHours,
      ...refundInfo,
    };
  }

  const allowed = Boolean(
    booking.CancellationAllowed ??
    booking.cancellation_allowed
  );

  if (!allowed) {
    return {
      eligible: false,
      reason: "Cancellation is not permitted for this event under its policy.",
      deadlineDate,
      deadlinePassed,
      deadlineHours,
      ...refundInfo,
    };
  }

  if (deadlinePassed) {
    return {
      eligible: false,
      reason: "The cancellation deadline for this event has passed.",
      deadlineDate,
      deadlinePassed: true,
      deadlineHours,
      ...refundInfo,
    };
  }

  if (Array.isArray(tickets) && tickets.length > 0) {
    const hasUsedTicket = tickets.some(
      (t) => (t.Status ?? t.status ?? "").toString().toUpperCase() === "USED"
    );
    if (hasUsedTicket) {
      return {
        eligible: false,
        reason: "Cannot cancel booking because one or more tickets have already been used.",
        deadlineDate,
        deadlinePassed,
        deadlineHours,
        ...refundInfo,
      };
    }
  }

  return {
    eligible: true,
    reason: null,
    deadlineDate,
    deadlinePassed: false,
    deadlineHours,
    ...refundInfo,
  };
}

/**
 * Formats a Date object into user-friendly cancellation deadline string.
 * @param {Date|string|null} date
 * @returns {string}
 */
export function formatDeadlineDateTime(date) {
  if (!date) return "Not specified";
  const d = date instanceof Date ? date : new Date(date);
  if (Number.isNaN(d.getTime())) return "Not specified";

  return d.toLocaleDateString("en-IN", {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
}
