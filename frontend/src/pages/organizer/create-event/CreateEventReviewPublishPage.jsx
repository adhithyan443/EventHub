import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";

import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import EventDetailsContent from "../../../components/organizer/events/EventDetailsContent";
import PrePublishChecklistCard from "../../../components/organizer/events/PrePublishChecklistCard";

import useEventCreationStore from "../../../store/eventCreationStore";

import {
  TICKET_MODES,
  ORGANIZER_ROUTES,
  isOnlineEvent,
  isPhysicalEvent,
  getEventDetailsRoute,
  parseAgeRestriction,
} from "../../../constants/eventConstants";

import {
  createOrganizerEvent,
  updateOrganizerEvent,
  publishOrganizerEvent,
} from "../../../api/organizerApi";

import notify from "../../../utils/toast";
import {
  isUUID,
  parseDeadlineHours,
  extractEventId,
  formatSubmissionErrorMessage,
  validateEventForDraft,
  validateEventForPublish,
} from "../../../utils/eventValidation";

import imgDefaultBanner from "../../../assets/organizer/0b2568d2a1321299cd93ab73936efb2e8bc467fe.png";

const getTotalSeatingCapacity = (categories) =>
  categories.reduce(
    (total, category) =>
      total +
      (category.rows || []).reduce(
        (rowTotal, row) => rowTotal + (row.seats?.length || 0),
        0
      ),
    0
  );

export default function CreateEventReviewPublishPage() {
  const navigate = useNavigate();

  const basicInformation = useEventCreationStore((state) => state.basicInformation);
  const locationType = useEventCreationStore((state) => state.locationType);
  const onlineUrl = useEventCreationStore((state) => state.onlineUrl);
  const venue = useEventCreationStore((state) => state.venue);
  const dateTime = useEventCreationStore((state) => state.dateTime);
  const eventDetails = useEventCreationStore((state) => state.eventDetails);
  const ticketMode = useEventCreationStore((state) => state.ticketMode);
  const ticketTypes = useEventCreationStore((state) => state.ticketTypes);
  const seatingConfig = useEventCreationStore((state) => state.seatingConfig);
  const ticketSalesSettings = useEventCreationStore((state) => state.ticketSalesSettings);
  const resetForm = useEventCreationStore((state) => state.resetForm);
  const isEditMode = useEventCreationStore((state) => state.isEditMode);
  const editingEventId = useEventCreationStore((state) => state.editingEventId);

  const [isPublishing, setIsPublishing] = useState(false);
  const [isSubmittingDraft, setIsSubmittingDraft] = useState(false);
  const [submitError, setSubmitError] = useState("");

  // Retain created event ID across attempts to prevent duplicate creations
  const [savedDraftEventId, setSavedDraftEventId] = useState(null);
  const [publishFailedAfterCreation, setPublishFailedAfterCreation] = useState(false);

  const isOnline = isOnlineEvent(locationType);
  const isPhysical = isPhysicalEvent(locationType);

  const showSeatedLayout = !isOnline && ticketMode === TICKET_MODES.SEATED;
  const seatingCategories = useMemo(
    () => seatingConfig?.categories || [],
    [seatingConfig?.categories]
  );

  const totalCapacity = useMemo(() => {
    if (showSeatedLayout) {
      return getTotalSeatingCapacity(seatingCategories);
    }
    return ticketTypes.reduce(
      (sum, ticket) => sum + (Number(ticket.capacity) || 0),
      0
    );
  }, [showSeatedLayout, seatingCategories, ticketTypes]);

  // Construct preview event object matching EventDetailsPage.jsx data shape
  const previewEvent = useMemo(() => {
    const cancellation = eventDetails?.cancellationPolicy || {};
    const cancellationDeadlineHours = parseDeadlineHours(
      cancellation.cancellationDeadline
    );

    const mappedTicketTypes = showSeatedLayout
      ? seatingCategories.map((cat, idx) => {
          const catCapacity = (cat.rows || []).reduce(
            (total, row) => total + (row.seats?.length || 0),
            0
          );
          return {
            ID: cat.id || `preview-cat-${idx}`,
            Name: cat.name || `Section ${idx + 1}`,
            Description: `${(cat.rows || []).length} rows configured`,
            Price: Number(String(cat.price ?? "").replace(/[₹,\s]/g, "") || 0),
            TotalQuantity: catCapacity,
            SoldQuantity: 0,
            AvailableQuantity: catCapacity,
            Status: "AVAILABLE",
          };
        })
      : ticketTypes.map((ticket, idx) => ({
          ID: ticket.id || `preview-ticket-${idx}`,
          Name: ticket.name || "General Admission",
          Description: ticket.description || "",
          Price: Number(ticket.price || 0),
          TotalQuantity: Number(ticket.capacity || 0),
          SoldQuantity: 0,
          AvailableQuantity: Number(ticket.capacity || 0),
          Status: "AVAILABLE",
        }));

    const mappedSeatLayout = showSeatedLayout
      ? {
          LayoutName: seatingConfig?.layoutName || "Main Seating Layout",
          Sections: seatingCategories.map((category, cIdx) => ({
            ID: category.id || `preview-section-${cIdx}`,
            Name: category.name || `Section ${cIdx + 1}`,
            Price: Number(
              String(category.price ?? "").replace(/[₹,\s]/g, "") || 0
            ),
            Rows: (category.rows || []).map((row, rIdx) => ({
              ID: row.id || `preview-row-${cIdx}-${rIdx}`,
              RowName: row.rowLetter || row.name || String.fromCharCode(65 + rIdx),
              Seats: (row.seats || []).map((seat, sIdx) => ({
                ID: seat.id || `preview-seat-${cIdx}-${rIdx}-${sIdx}`,
                SeatNumber: seat.seatNumber || sIdx + 1,
                Status: seat.status || "AVAILABLE",
              })),
            })),
          })),
        }
      : null;

    return {
      Event: {
        Title: basicInformation.title || "Untitled Event",
        Description: basicInformation.description || "No description provided.",
        CategoryName: basicInformation.category || "Uncategorized",
        BannerImageURL:
          basicInformation.bannerPreviewUrl ||
          basicInformation.banner ||
          imgDefaultBanner,
        AgeRestriction: parseAgeRestriction(basicInformation.ageRestriction),
        Visibility: eventDetails.visibility || "PUBLIC",
        EventType: locationType || "PHYSICAL",
        OnlineURL: onlineUrl || "",
        Highlights: eventDetails.highlights || "",
        Rules: eventDetails.rules || "",
        AttendeeInformation: eventDetails.attendeeInformation || "",
        Status: "DRAFT",
        ID:
          savedDraftEventId ||
          (isEditMode && editingEventId ? editingEventId : "Draft Preview"),
        CreatedAt: new Date().toISOString(),
      },
      Schedule: {
        EventDate: dateTime.eventDate || "",
        StartTime: dateTime.startTime || "",
        EndTime: dateTime.endTime || "",
        IsAllDay: Boolean(dateTime.isAllDay),
      },
      Venue: {
        Name: venue?.name || (isOnline ? "Online Event" : ""),
        Address: venue?.address || "",
        City: venue?.city || (isOnline ? "Virtual Stream" : ""),
        State: venue?.state || "",
        Country: venue?.country || "",
        PostalCode: venue?.postal_code || "",
      },
      Setting: {
        SeatLayoutType: showSeatedLayout ? "SEATED" : "GENERAL",
        BookingLimitPerUser: Number(ticketSalesSettings.maxTicketsPerBooking || 1),
        SalesStartDate: ticketSalesSettings.salesStartDate || "",
        SalesEndDate: ticketSalesSettings.salesEndDate || "",
      },
      Cancellation: {
        CancellationAllowed: Boolean(cancellation.allowCancellation),
        CancellationDeadlineHours: cancellationDeadlineHours,
        RefundPolicy: cancellation.refundPolicy || "NO_REFUND",
        RefundPercentage: Number(cancellation.refundPercentage || 0),
      },
      Contact: {
        Name: eventDetails.contactInformation?.name || "",
        Phone: eventDetails.contactInformation?.phone || "",
        Email: eventDetails.contactInformation?.email || "",
      },
      TicketTypes: mappedTicketTypes,
      SeatLayout: mappedSeatLayout,
      Statistics: {
        Revenue: 0,
        TicketCapacity: totalCapacity,
        TicketsSold: 0,
        PercentageSold: 0,
        IsMock: false,
      },
    };
  }, [
    basicInformation,
    locationType,
    onlineUrl,
    venue,
    dateTime,
    eventDetails,
    showSeatedLayout,
    seatingCategories,
    ticketTypes,
    seatingConfig,
    ticketSalesSettings,
    totalCapacity,
    savedDraftEventId,
    isEditMode,
    editingEventId,
    isOnline,
  ]);

  const buildEventFormData = () => {
    const cancellationAllowed = Boolean(
      eventDetails?.cancellationPolicy?.allowCancellation
    );
    const deadlineHours = cancellationAllowed
      ? parseDeadlineHours(eventDetails?.cancellationPolicy?.cancellationDeadline)
      : 0;

    const ageRestriction = parseAgeRestriction(basicInformation.ageRestriction);

    const eventPayload = {
      category_id: basicInformation.categoryId,
      event_type: locationType,
      online_url: isPhysical ? "" : onlineUrl.trim(),
      title: basicInformation.title.trim(),
      description: basicInformation.description.trim(),
      banner_url: "",
      language: basicInformation.language.trim(),
      age_restriction: ageRestriction,
      visibility: eventDetails.visibility || "PUBLIC",
      highlights: eventDetails.highlights?.trim() || "",
      rules: eventDetails.rules?.trim() || "",
      attendee_information: eventDetails.attendeeInformation?.trim() || "",
      contact: {
        name: eventDetails.contactInformation?.name?.trim() || "",
        phone: eventDetails.contactInformation?.phone?.trim() || "",
        email: eventDetails.contactInformation?.email?.trim() || "",
      },
      event_date: dateTime.eventDate,
      start_time: dateTime.isAllDay ? "" : dateTime.startTime,
      end_time: dateTime.isAllDay ? "" : dateTime.endTime,
      is_all_day: Boolean(dateTime.isAllDay),
      sales_start_date: ticketSalesSettings.salesStartDate || "",
      sales_end_date: ticketSalesSettings.salesEndDate || "",
      seat_layout_type: showSeatedLayout ? "SEATED" : "GENERAL",
      ticket_types: showSeatedLayout
        ? []
        : ticketTypes.map((ticket) => ({
            ...(isEditMode && isUUID(ticket.id) ? { id: ticket.id } : {}),
            name: ticket.name.trim(),
            price: Number(ticket.price),
            capacity: Number(ticket.capacity),
            description: ticket.description?.trim() || "",
          })),
      ...(!isEditMode &&
        showSeatedLayout && {
          seat_layout: {
            layout_name: seatingConfig?.layoutName || "Main Seating Layout",
            sections: seatingCategories.map((category) => ({
              name: category.name.trim(),
              price: Number(
                String(category.price ?? "").replace(/[₹,\s]/g, "")
              ),
              rows: category.rows.map((row) => ({
                row_name: row.rowLetter?.trim() || row.name?.trim(),
                seats: row.seats.length,
              })),
            })),
          },
        }),
      booking_limit_per_user: Number(
        ticketSalesSettings.maxTicketsPerBooking || 1
      ),
      cancellation_allowed: cancellationAllowed,
      cancellation_deadline_hours: deadlineHours,
      refund_policy: cancellationAllowed
        ? eventDetails.cancellationPolicy.refundPolicy
        : "",
      refund_percentage: cancellationAllowed
        ? Number(eventDetails.cancellationPolicy.refundPercentage || 0)
        : 0,
      venue: isOnline
        ? {
            google_place_id: "",
            name: "",
            address: "",
            city: "",
            state: "",
            country: "",
            postal_code: "",
            latitude: 0,
            longitude: 0,
          }
        : {
            google_place_id: venue.google_place_id?.trim() || "",
            name: venue.name?.trim() || "",
            address: venue.address?.trim() || "",
            city: venue.city?.trim() || "",
            state: venue.state?.trim() || "",
            country: venue.country?.trim() || "",
            postal_code: venue.postal_code?.trim() || "",
            latitude: Number(venue.latitude) || 0,
            longitude: Number(venue.longitude) || 0,
          },
    };

    const formData = new FormData();
    formData.append("event", JSON.stringify(eventPayload));
    if (basicInformation.bannerFile) {
      formData.append("banner", basicInformation.bannerFile);
    }
    return formData;
  };

  const handleBack = () => {
    if (showSeatedLayout) {
      navigate(ORGANIZER_ROUTES.CREATE_SEAT_CONFIG);
    } else {
      navigate(ORGANIZER_ROUTES.CREATE_TICKET_TYPES);
    }
  };

  // Action 1: Save as Draft
  const handleSaveAsDraft = async () => {
    if (isPublishing || isSubmittingDraft) return;

    // Apply validation genuinely required to persist the draft event
    const validationError = validateEventForDraft({
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
      isEditMode,
    });

    if (validationError) {
      setSubmitError(validationError);
      notify.error(validationError);
      return;
    }

    try {
      setIsSubmittingDraft(true);
      setSubmitError("");

      let targetEventId = savedDraftEventId || (isEditMode ? editingEventId : null);

      if (targetEventId) {
        // If event was already created during a prior attempt or is edit mode, update draft
        const formData = buildEventFormData();
        await updateOrganizerEvent(targetEventId, formData);
      } else {
        const formData = buildEventFormData();
        const response = await createOrganizerEvent(formData);
        targetEventId = extractEventId(response);

        if (!targetEventId) {
          throw new Error("Unable to retrieve ID for created draft.");
        }

        setSavedDraftEventId(targetEventId);
      }

      resetForm();
      notify.success("Event saved as draft successfully!");
      navigate(getEventDetailsRoute(targetEventId), {
        state: {
          successMessage: "Event saved as draft successfully!",
        },
      });
    } catch (err) {
      console.error("Failed to save draft:", err);
      const errMsg = formatSubmissionErrorMessage(
        err,
        "Failed to save draft. Please check your inputs and try again."
      );
      setSubmitError(errMsg);
      notify.error(errMsg);
    } finally {
      setIsSubmittingDraft(false);
    }
  };

  // Action 2: Publish Event
  const handlePublish = async () => {
    if (isPublishing || isSubmittingDraft) return;

    // Apply all publication-readiness requirements in addition to draft validity
    const validationError = validateEventForPublish({
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
      isEditMode,
    });

    if (validationError) {
      setSubmitError(validationError);
      notify.error(validationError);
      return;
    }

    try {
      setIsPublishing(true);
      setSubmitError("");

      let targetEventId = savedDraftEventId || (isEditMode ? editingEventId : null);

      // Step 2.1: If event was not yet created, create it as draft first
      if (!targetEventId) {
        const formData = buildEventFormData();
        const createRes = await createOrganizerEvent(formData);
        targetEventId = extractEventId(createRes);

        if (!targetEventId) {
          throw new Error("Unable to retrieve event ID from creation response.");
        }

        // Retain ID so retry will NEVER create a duplicate record
        setSavedDraftEventId(targetEventId);
      } else if (isEditMode && !savedDraftEventId) {
        // If edit mode and not yet updated in this step, update details
        const formData = buildEventFormData();
        await updateOrganizerEvent(targetEventId, formData);
      }

      // Step 2.2: Immediately transition status from DRAFT to PUBLISHED
      try {
        await publishOrganizerEvent(targetEventId);

        // Success: Reset creation wizard store and navigate
        resetForm();
        notify.success("Event published live successfully!");
        navigate(getEventDetailsRoute(targetEventId), {
          state: {
            successMessage: "Event published live successfully!",
          },
        });
      } catch (pubErr) {
        console.error("Publishing step failed after event creation:", pubErr);
        const pubErrMsg = formatSubmissionErrorMessage(
          pubErr,
          "Event draft was saved, but publishing failed. You can retry publishing below."
        );

        setPublishFailedAfterCreation(true);
        setSubmitError(
          `Event draft was saved successfully (ID: ${targetEventId}), but publishing failed: ${pubErrMsg}`
        );
        notify.error(pubErrMsg);
      }
    } catch (err) {
      console.error("Failed to create event:", err);
      const errMsg = formatSubmissionErrorMessage(
        err,
        "Failed to create event. Please try again."
      );
      setSubmitError(errMsg);
      notify.error(errMsg);
    } finally {
      setIsPublishing(false);
    }
  };

  return (
    <div className="flex flex-col min-h-screen bg-[#f9f9ff]">
      <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex flex-col gap-6 w-full flex-1">
        {/* Step Navigation Header */}
        <div className="flex flex-col gap-4">
          <EventCreationStepper currentStep={5} />

          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-3">
              <h1 className="text-xl sm:text-2xl font-bold text-[#141b2b]">
                {isEditMode ? "Review & Save Changes" : "Review & Publish"}
              </h1>
              {isEditMode && (
                <span className="text-xs font-semibold px-2.5 py-0.5 rounded-full bg-amber-50 text-amber-700 border border-amber-200">
                  Edit Mode
                </span>
              )}
            </div>

            <p className="text-xs text-[#565e74]">
              {isEditMode
                ? "Review your changes before updating this event."
                : "Review all event details, ticket configurations, and location settings before publishing live."}
            </p>
          </div>
        </div>

        {/* Partial Success / Error Banner */}
        {publishFailedAfterCreation && savedDraftEventId && (
          <div className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-amber-900 shadow-xs flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 animate-fade-in">
            <div className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <span className="font-bold text-sm">Draft Saved Successfully</span>
                <span className="text-xs bg-amber-200/80 text-amber-900 px-2 py-0.5 rounded font-mono">
                  {savedDraftEventId}
                </span>
              </div>
              <p className="text-xs text-amber-800 leading-relaxed">
                Your event draft is securely saved in the database, but publication could not be completed:{" "}
                <span className="font-medium">{submitError}</span>. You can click <strong>Publish Event</strong> below to retry publishing without re-creating the event.
              </p>
            </div>

            <button
              type="button"
              onClick={() => {
                resetForm();
                navigate(getEventDetailsRoute(savedDraftEventId));
              }}
              className="whitespace-nowrap px-3.5 py-2 bg-amber-700 hover:bg-amber-800 text-white rounded-lg text-xs font-semibold cursor-pointer transition-colors shrink-0 shadow-2xs"
            >
              View Saved Draft &rarr;
            </button>
          </div>
        )}

        {submitError && !publishFailedAfterCreation && (
          <div className="rounded-lg border border-[#ba1a1a]/30 bg-[#ba1a1a]/5 px-4 py-3 text-sm text-[#ba1a1a]">
            {submitError}
          </div>
        )}

        {/* Reused Event Details Preview Content */}
        <EventDetailsContent
          event={previewEvent}
          isPreview={true}
          rightColumnExtra={
            <PrePublishChecklistCard
              hasBasicInfo={Boolean(
                basicInformation.title?.trim() &&
                  basicInformation.categoryId &&
                  basicInformation.description?.trim()
              )}
              hasVenueOrUrl={Boolean(
                isOnline
                  ? onlineUrl?.trim()
                  : venue.name?.trim() &&
                      venue.address?.trim() &&
                      venue.city?.trim() &&
                      venue.state?.trim() &&
                      venue.country?.trim() &&
                      venue.google_place_id?.trim()
              )}
              hasSchedule={Boolean(dateTime.eventDate)}
              hasTickets={Boolean(totalCapacity > 0)}
              hasCancellation={Boolean(
                eventDetails?.cancellationPolicy?.organizerPolicyAccepted
              )}
              visibility={eventDetails.visibility || "PUBLIC"}
            />
          }
        />
      </div>

      {/* Sticky Action Footer */}
      <EventCreationFooter
        onBack={handleBack}
        onContinue={handlePublish}
        onSaveDraft={handleSaveAsDraft}
        isReviewStep={true}
        isPublishing={isPublishing}
        isSubmittingDraft={isSubmittingDraft}
        continueLabel={
          isPublishing
            ? "Publishing Event..."
            : isEditMode
            ? "🚀 Publish Updated Event"
            : "🚀 Publish Event"
        }
        saveDraftLabel={isSubmittingDraft ? "Saving..." : "Save as Draft"}
        disabled={isPublishing || isSubmittingDraft}
      />
    </div>
  );
}
