import { useState, useEffect } from "react";
import { getAdminEventById } from "../../api/adminEventApi";
import notify from "../../utils/toast";

const statusBadgeStyles = {
  PUBLISHED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  DRAFT: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
};

export default function AdminEventDetailDrawer({
  event,
  onClose,
  onEditClick,
  onPublishClick,
  onDeleteClick,
}) {
  const [details, setDetails] = useState(null);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState("overview"); // overview, organizer, ticketing, metrics
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const loadDetails = async () => {
      if (!event?.id) return;
      try {
        setLoading(true);
        const res = await getAdminEventById(event.id);
        if (!cancelled && res?.data) {
          setDetails(res.data);
        }
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to fetch fresh event details:", err);
          notify.error("Failed to load full event details.");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadDetails();

    return () => {
      cancelled = true;
    };
  }, [event]);

  if (!event) return null;

  const currentEvent = details?.event || event;
  const organizer = details?.organizer;
  const category = details?.category;
  const schedule = details?.schedule;
  const venue = details?.venue;
  const setting = details?.setting;
  const cancellation = details?.cancellation;
  const contact = details?.contact;
  const ticketTypes = Array.isArray(details?.ticket_types) ? details.ticket_types : [];
  const seatLayout = details?.seat_layout;
  const seatSections = Array.isArray(details?.seat_sections) ? details.seat_sections : [];
  const bookingStats = details?.booking_stats || {
    total_bookings: 0,
    confirmed_bookings: 0,
    cancelled_bookings: 0,
    tickets_sold: 0,
    total_revenue: "0",
  };
  const activeReservations = details?.active_reservations || 0;

  const status = currentEvent.status?.toUpperCase() || "DRAFT";
  const statusBadgeClass =
    statusBadgeStyles[status] || "bg-[#f1f3ff] text-[#141b2b]";

  const bannerSrc =
    details?.banner_image_url ||
    currentEvent.banner_image_url ||
    currentEvent.banner_url ||
    "";

  const handleCopyId = () => {
    if (navigator.clipboard && currentEvent.id) {
      navigator.clipboard.writeText(currentEvent.id);
      setCopied(true);
      notify.success("Event ID copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const formatDate = (dateStr) => {
    if (!dateStr) return "—";
    const d = new Date(dateStr);
    return isNaN(d.getTime())
      ? "—"
      : d.toLocaleDateString("en-IN", {
          day: "numeric",
          month: "short",
          year: "numeric",
        });
  };

  const formatTime = (timeStr) => {
    if (!timeStr) return "—";
    const d = new Date(timeStr);
    return isNaN(d.getTime())
      ? timeStr
      : d.toLocaleTimeString("en-IN", {
          hour: "2-digit",
          minute: "2-digit",
        });
  };

  const formatCurrency = (amount) => {
    const num = parseFloat(amount || 0);
    return new Intl.NumberFormat("en-IN", {
      style: "currency",
      currency: "INR",
      maximumFractionDigits: 0,
    }).format(num);
  };

  return (
    <div className="fixed inset-0 z-40 flex justify-end animate-in fade-in duration-200">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={onClose}
      />

      {/* Slide Drawer */}
      <div className="relative w-full max-w-[560px] bg-white shadow-2xl flex flex-col h-full overflow-hidden z-10 animate-in slide-in-from-right duration-200">
        {/* Drawer Header */}
        <div className="p-5 border-b border-[#dce2f7] flex items-center justify-between shrink-0 bg-[#f9f9ff]">
          <div className="flex items-center gap-3 min-w-0 pr-2">
            <div className="w-10 h-10 rounded-lg bg-[#eef0fb] text-[#141b2b] flex items-center justify-center font-bold text-lg shrink-0">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                <line x1="16" y1="2" x2="16" y2="6" />
                <line x1="8" y1="2" x2="8" y2="6" />
                <line x1="3" y1="10" x2="21" y2="10" />
              </svg>
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <h2 className="text-[#141b2b] text-base font-bold truncate">
                  {currentEvent.title || "Event Details"}
                </h2>
                <span
                  className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${statusBadgeClass}`}
                >
                  {status}
                </span>
              </div>
              <p className="text-xs text-[#3d4947] truncate">
                {category?.name || currentEvent.category_name || "Event"} •{" "}
                {currentEvent.event_type || "PHYSICAL"}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center text-[#3d4947] hover:bg-[#eef0fb] transition-colors cursor-pointer shrink-0"
            aria-label="Close"
          >
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path
                d="M1 1L13 13M1 13L13 1"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              />
            </svg>
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-[#dce2f7] bg-white px-5 shrink-0 overflow-x-auto text-xs font-semibold">
          <button
            type="button"
            onClick={() => setActiveTab("overview")}
            className={`py-3 px-3 border-b-2 transition-colors cursor-pointer shrink-0 ${
              activeTab === "overview"
                ? "border-[#141b2b] text-[#141b2b]"
                : "border-transparent text-[#3d4947] hover:text-[#141b2b]"
            }`}
          >
            Overview
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("organizer")}
            className={`py-3 px-3 border-b-2 transition-colors cursor-pointer shrink-0 ${
              activeTab === "organizer"
                ? "border-[#141b2b] text-[#141b2b]"
                : "border-transparent text-[#3d4947] hover:text-[#141b2b]"
            }`}
          >
            Organizer
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("ticketing")}
            className={`py-3 px-3 border-b-2 transition-colors cursor-pointer shrink-0 ${
              activeTab === "ticketing"
                ? "border-[#141b2b] text-[#141b2b]"
                : "border-transparent text-[#3d4947] hover:text-[#141b2b]"
            }`}
          >
            Ticketing & Seating
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("metrics")}
            className={`py-3 px-3 border-b-2 transition-colors cursor-pointer shrink-0 ${
              activeTab === "metrics"
                ? "border-[#141b2b] text-[#141b2b]"
                : "border-transparent text-[#3d4947] hover:text-[#141b2b]"
            }`}
          >
            Booking Metrics
          </button>
        </div>

        {/* Drawer Scrollable Body */}
        <div className="flex-1 overflow-y-auto p-5 space-y-6">
          {loading && !details ? (
            <div className="py-12 flex flex-col items-center justify-center text-[#3d4947] gap-3">
              <div className="w-8 h-8 border-3 border-[#141b2b] border-t-transparent rounded-full animate-spin" />
              <p className="text-xs font-medium">Loading event details...</p>
            </div>
          ) : (
            <>
              {/* Event Banner */}
              {bannerSrc ? (
                <div className="w-full h-44 rounded-xl overflow-hidden bg-slate-100 border border-[#dce2f7] relative">
                  <img
                    src={bannerSrc}
                    alt={currentEvent.title}
                    className="w-full h-full object-cover"
                    onError={(e) => {
                      e.target.style.display = "none";
                    }}
                  />
                </div>
              ) : (
                <div className="w-full h-24 rounded-xl bg-[#f9f9ff] border border-dashed border-[#bcc9c6] flex items-center justify-center text-xs text-[#3d4947]">
                  No event banner uploaded
                </div>
              )}

              {/* TAB 1: OVERVIEW */}
              {activeTab === "overview" && (
                <div className="space-y-5">
                  {/* Event ID copy row */}
                  <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-3 flex items-center justify-between text-xs">
                    <div>
                      <span className="text-[#3d4947] font-medium">Event ID: </span>
                      <span className="font-mono text-[#141b2b]">{currentEvent.id}</span>
                    </div>
                    <button
                      type="button"
                      onClick={handleCopyId}
                      className="text-[#00685f] hover:underline font-semibold cursor-pointer shrink-0"
                    >
                      {copied ? "Copied!" : "Copy"}
                    </button>
                  </div>

                  {/* Schedule & Venue Card */}
                  <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-3">
                    <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                      Schedule & Location
                    </h3>
                    <div className="grid grid-cols-2 gap-3 text-xs">
                      <div>
                        <span className="text-[#3d4947] block">Event Date:</span>
                        <span className="font-semibold text-[#141b2b]">
                          {formatDate(schedule?.event_date || currentEvent.event_date)}
                        </span>
                      </div>
                      <div>
                        <span className="text-[#3d4947] block">Time:</span>
                        <span className="font-semibold text-[#141b2b]">
                          {schedule?.is_all_day || currentEvent.is_all_day
                            ? "All-Day Event"
                            : `${formatTime(schedule?.start_time || currentEvent.start_time)} - ${formatTime(schedule?.end_time || currentEvent.end_time)}`}
                        </span>
                      </div>
                    </div>

                    <div className="pt-2 border-t border-[#f1f3ff] text-xs">
                      <span className="text-[#3d4947] block">Location:</span>
                      {currentEvent.event_type === "ONLINE" ? (
                        <span className="font-semibold text-[#00685f] break-all">
                          {currentEvent.online_url || "Online Stream"}
                        </span>
                      ) : (
                        <div className="font-medium text-[#141b2b]">
                          <div className="font-semibold text-sm">
                            {venue?.name || currentEvent.venue_name || "Physical Venue"}
                          </div>
                          <div className="text-[#3d4947]">
                            {venue?.address || ""}{venue?.city ? `, ${venue.city}` : ""}{venue?.state ? `, ${venue.state}` : ""}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Description */}
                  <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-2">
                    <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                      Description
                    </h3>
                    <p className="text-xs text-[#3d4947] leading-relaxed whitespace-pre-line">
                      {currentEvent.description || "No description provided."}
                    </p>
                  </div>

                  {/* Highlights & Rules */}
                  {(currentEvent.highlights || currentEvent.rules || currentEvent.attendee_information) && (
                    <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-3 text-xs">
                      {currentEvent.highlights && (
                        <div>
                          <span className="font-bold text-[#141b2b] block mb-1">Highlights:</span>
                          <p className="text-[#3d4947] whitespace-pre-line">{currentEvent.highlights}</p>
                        </div>
                      )}
                      {currentEvent.rules && (
                        <div>
                          <span className="font-bold text-[#141b2b] block mb-1">Rules & Guidelines:</span>
                          <p className="text-[#3d4947] whitespace-pre-line">{currentEvent.rules}</p>
                        </div>
                      )}
                      {currentEvent.attendee_information && (
                        <div>
                          <span className="font-bold text-[#141b2b] block mb-1">Attendee Information:</span>
                          <p className="text-[#3d4947] whitespace-pre-line">{currentEvent.attendee_information}</p>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Contact Information */}
                  {contact && (
                    <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-2 text-xs">
                      <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                        Event Contact
                      </h3>
                      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                        <div>
                          <span className="text-[#3d4947] block">Name:</span>
                          <span className="font-semibold text-[#141b2b]">{contact.name || "—"}</span>
                        </div>
                        <div>
                          <span className="text-[#3d4947] block">Email:</span>
                          <span className="font-semibold text-[#141b2b]">{contact.email || "—"}</span>
                        </div>
                        <div>
                          <span className="text-[#3d4947] block">Phone:</span>
                          <span className="font-semibold text-[#141b2b]">{contact.phone || "—"}</span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* TAB 2: ORGANIZER */}
              {activeTab === "organizer" && (
                <div className="space-y-4">
                  <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-3 text-xs">
                    <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                      Organizer Profile
                    </h3>
                    <div className="space-y-2">
                      <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                        <span className="text-[#3d4947]">Business Name:</span>
                        <span className="font-bold text-[#141b2b]">
                          {organizer?.business_name || currentEvent.organizer_business_name || "—"}
                        </span>
                      </div>
                      <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                        <span className="text-[#3d4947]">Business Type:</span>
                        <span className="font-medium text-[#141b2b]">
                          {organizer?.business_type || "—"}
                        </span>
                      </div>
                      <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                        <span className="text-[#3d4947]">Organizer Email:</span>
                        <span className="font-medium text-[#141b2b]">
                          {organizer?.email || currentEvent.organizer_email || "—"}
                        </span>
                      </div>
                      <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                        <span className="text-[#3d4947]">Phone:</span>
                        <span className="font-medium text-[#141b2b]">
                          {organizer?.phone || "—"}
                        </span>
                      </div>
                      <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                        <span className="text-[#3d4947]">Account Status:</span>
                        <span className="font-semibold text-[#00685f]">
                          {organizer?.status || "ACTIVE"}
                        </span>
                      </div>
                      {organizer?.website && (
                        <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                          <span className="text-[#3d4947]">Website:</span>
                          <a
                            href={organizer.website}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-[#00685f] hover:underline truncate max-w-[200px]"
                          >
                            {organizer.website}
                          </a>
                        </div>
                      )}
                      {organizer?.gst_number && (
                        <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                          <span className="text-[#3d4947]">GST Number:</span>
                          <span className="font-mono text-[#141b2b]">{organizer.gst_number}</span>
                        </div>
                      )}
                      {organizer?.pan_number && (
                        <div className="flex justify-between py-1">
                          <span className="text-[#3d4947]">PAN Number:</span>
                          <span className="font-mono text-[#141b2b]">{organizer.pan_number}</span>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              )}

              {/* TAB 3: TICKETING & SEATING */}
              {activeTab === "ticketing" && (
                <div className="space-y-4">
                  {/* Settings overview */}
                  <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-2 text-xs">
                    <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider mb-2">
                      Ticketing Configuration
                    </h3>
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <span className="text-[#3d4947] block">Layout Mode:</span>
                        <span className="font-bold text-[#141b2b]">
                          {setting?.seat_layout_type || currentEvent.seat_layout_type || "GENERAL"}
                        </span>
                      </div>
                      <div>
                        <span className="text-[#3d4947] block">Booking Limit / User:</span>
                        <span className="font-semibold text-[#141b2b]">
                          {setting?.booking_limit_per_user || 10} tickets
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Cancellation Settings */}
                  {cancellation && (
                    <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-2 text-xs">
                      <h3 className="text-xs font-bold text-[#141b2b] uppercase tracking-wider mb-2">
                        Cancellation Policy
                      </h3>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <span className="text-[#3d4947] block">Cancellation Allowed:</span>
                          <span className="font-semibold text-[#141b2b]">
                            {cancellation.cancellation_allowed ? "Yes" : "No"}
                          </span>
                        </div>
                        <div>
                          <span className="text-[#3d4947] block">Refund Policy:</span>
                          <span className="font-semibold text-[#141b2b]">
                            {cancellation.refund_policy || "NO_REFUND"} ({cancellation.refund_percentage}%)
                          </span>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Seated layout sections */}
                  {setting?.seat_layout_type === "SEATED" ? (
                    <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-3 text-xs">
                      <div className="flex items-center justify-between">
                        <h3 className="font-bold text-[#141b2b] uppercase tracking-wider">
                          Seat Layout: {seatLayout?.layout_name || "Seated Layout"}
                        </h3>
                        <span className="text-[10px] font-semibold bg-[#eef0fb] text-[#141b2b] px-2 py-0.5 rounded-full">
                          {seatSections.length} Sections
                        </span>
                      </div>

                      {seatSections.length === 0 ? (
                        <p className="text-[#3d4947] italic">No seat sections configured.</p>
                      ) : (
                        <div className="overflow-x-auto">
                          <table className="w-full text-left border-collapse">
                            <thead>
                              <tr className="border-b border-[#dce2f7] text-[#3d4947] text-[11px]">
                                <th className="py-2 pr-2">Section</th>
                                <th className="py-2 px-2">Price</th>
                                <th className="py-2 px-2">Capacity</th>
                                <th className="py-2 px-2">Sold</th>
                                <th className="py-2 pl-2">Available</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-[#f1f3ff]">
                              {seatSections.map((sec) => (
                                <tr key={sec.id}>
                                  <td className="py-2 pr-2 font-medium text-[#141b2b]">{sec.name}</td>
                                  <td className="py-2 px-2 font-mono">{formatCurrency(sec.price)}</td>
                                  <td className="py-2 px-2">{sec.total_capacity}</td>
                                  <td className="py-2 px-2 font-semibold text-[#00685f]">{sec.sold_quantity}</td>
                                  <td className="py-2 pl-2 font-semibold text-[#141b2b]">{sec.available_quantity}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  ) : (
                    /* General Admission Ticket Types */
                    <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-3 text-xs">
                      <div className="flex items-center justify-between">
                        <h3 className="font-bold text-[#141b2b] uppercase tracking-wider">
                          Ticket Types
                        </h3>
                        <span className="text-[10px] font-semibold bg-[#eef0fb] text-[#141b2b] px-2 py-0.5 rounded-full">
                          {ticketTypes.length} Tiers
                        </span>
                      </div>

                      {ticketTypes.length === 0 ? (
                        <p className="text-[#3d4947] italic">No ticket types configured.</p>
                      ) : (
                        <div className="overflow-x-auto">
                          <table className="w-full text-left border-collapse">
                            <thead>
                              <tr className="border-b border-[#dce2f7] text-[#3d4947] text-[11px]">
                                <th className="py-2 pr-2">Tier Name</th>
                                <th className="py-2 px-2">Price</th>
                                <th className="py-2 px-2">Total</th>
                                <th className="py-2 pl-2">Available</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-[#f1f3ff]">
                              {ticketTypes.map((tt) => (
                                <tr key={tt.id}>
                                  <td className="py-2 pr-2 font-medium text-[#141b2b]">{tt.name}</td>
                                  <td className="py-2 px-2 font-mono">{formatCurrency(tt.price)}</td>
                                  <td className="py-2 px-2">{tt.total_quantity}</td>
                                  <td className="py-2 pl-2 font-semibold text-[#00685f]">
                                    {tt.available_quantity}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}

              {/* TAB 4: METRICS */}
              {activeTab === "metrics" && (
                <div className="space-y-4">
                  <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-3 text-[11px] text-[#3d4947] flex items-center gap-2">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="12" y1="16" x2="12" y2="12" />
                      <line x1="12" y1="8" x2="12.01" y2="8" />
                    </svg>
                    <span>Accurate platform booking metrics without exposing customer PII.</span>
                  </div>

                  <div className="grid grid-cols-2 gap-3 text-xs">
                    <div className="border border-[#dce2f7] rounded-xl p-3 bg-white">
                      <span className="text-[#3d4947] text-[10px] uppercase font-bold tracking-wider block mb-1">
                        Confirmed Bookings
                      </span>
                      <span className="text-xl font-bold text-[#141b2b]">
                        {bookingStats.confirmed_bookings}
                      </span>
                    </div>

                    <div className="border border-[#dce2f7] rounded-xl p-3 bg-white">
                      <span className="text-[#3d4947] text-[10px] uppercase font-bold tracking-wider block mb-1">
                        Tickets Sold
                      </span>
                      <span className="text-xl font-bold text-[#00685f]">
                        {bookingStats.tickets_sold}
                      </span>
                    </div>

                    <div className="border border-[#dce2f7] rounded-xl p-3 bg-white">
                      <span className="text-[#3d4947] text-[10px] uppercase font-bold tracking-wider block mb-1">
                        Total Gross Revenue
                      </span>
                      <span className="text-lg font-bold text-[#141b2b] font-mono">
                        {formatCurrency(bookingStats.total_revenue)}
                      </span>
                    </div>

                    <div className="border border-[#dce2f7] rounded-xl p-3 bg-white">
                      <span className="text-[#3d4947] text-[10px] uppercase font-bold tracking-wider block mb-1">
                        Active Holds (Reservations)
                      </span>
                      <span className="text-xl font-bold text-[#eab308]">
                        {activeReservations}
                      </span>
                    </div>

                    <div className="border border-[#dce2f7] rounded-xl p-3 bg-white">
                      <span className="text-[#3d4947] text-[10px] uppercase font-bold tracking-wider block mb-1">
                        Total Orders
                      </span>
                      <span className="text-xl font-bold text-[#141b2b]">
                        {bookingStats.total_bookings}
                      </span>
                    </div>

                    <div className="border border-[#dce2f7] rounded-xl p-3 bg-white">
                      <span className="text-[#3d4947] text-[10px] uppercase font-bold tracking-wider block mb-1">
                        Cancelled Orders
                      </span>
                      <span className="text-xl font-bold text-[#ba1a1a]">
                        {bookingStats.cancelled_bookings}
                      </span>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Drawer Actions Footer (Strictly NO Cancel Action) */}
        <div className="p-4 border-t border-[#dce2f7] bg-[#f9f9ff] flex items-center justify-between gap-3 shrink-0">
          <button
            type="button"
            onClick={() => onDeleteClick(currentEvent)}
            className="text-[#ba1a1a] hover:bg-[#fff5f4] border border-[#ffdad6] text-xs font-semibold px-3 py-2 rounded-lg transition-colors cursor-pointer"
          >
            Delete Event
          </button>

          <div className="flex items-center gap-2">
            {status === "DRAFT" && (
              <button
                type="button"
                onClick={() => onPublishClick(currentEvent)}
                className="bg-[#00685f] hover:bg-[#005049] text-white text-xs font-semibold px-4 py-2 rounded-lg transition-colors cursor-pointer shadow-sm"
              >
                Publish Event
              </button>
            )}

            {status !== "CANCELLED" && (
              <button
                type="button"
                onClick={() => onEditClick(currentEvent)}
                className="bg-[#141b2b] hover:bg-[#2a3449] text-white text-xs font-semibold px-4 py-2 rounded-lg transition-colors cursor-pointer shadow-sm"
              >
                Edit Event
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
