import { useState, useEffect } from "react";
import { updateAdminEvent, getCategories } from "../../api/adminEventApi";
import notify from "../../utils/toast";

const buildInitialFormData = (ev) => ({
  title: ev?.title || "",
  description: ev?.description || "",
  banner_url: ev?.banner_url || "",
  language: ev?.language || "English",
  age_restriction: ev?.age_restriction || 0,
  highlights: ev?.highlights || "",
  rules: ev?.rules || "",
  attendee_information: ev?.attendee_information || "",

  contact_name: ev?.contact?.name || ev?.contact_name || "",
  contact_phone: ev?.contact?.phone || ev?.contact_phone || "",
  contact_email: ev?.contact?.email || ev?.contact_email || "",

  // Draft-only structural fields
  category_id: ev?.category_id || "",
  event_type: ev?.event_type || "PHYSICAL",
  online_url: ev?.online_url || "",
  event_date: ev?.event_date ? String(ev.event_date).split("T")[0] : "",
  start_time: ev?.start_time ? String(ev.start_time).substring(11, 16) : "",
  end_time: ev?.end_time ? String(ev.end_time).substring(11, 16) : "",
  is_all_day: Boolean(ev?.is_all_day),
  booking_limit_per_user: ev?.booking_limit_per_user || 10,
  cancellation_allowed: ev?.cancellation_allowed !== false,
  cancellation_deadline_hours: ev?.cancellation_deadline_hours || 24,
  refund_policy: ev?.refund_policy || "PARTIAL",
  refund_percentage: ev?.refund_percentage || 80,
});

export default function EditEventModal({
  event,
  isOpen,
  onClose,
  onSuccess,
}) {
  const [categories, setCategories] = useState([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const isPublished = event?.status?.toUpperCase() === "PUBLISHED";
  const isDraft = event?.status?.toUpperCase() === "DRAFT";

  // Form State
  const [formData, setFormData] = useState(() => buildInitialFormData(event));
  const [prevEventId, setPrevEventId] = useState(event?.id);

  // Sync state when event prop changes without calling setState inside an effect
  if (event?.id !== prevEventId) {
    setPrevEventId(event?.id);
    setFormData(buildInitialFormData(event));
    setError("");
  }

  useEffect(() => {
    let cancelled = false;

    getCategories()
      .then((res) => {
        if (cancelled) return;
        const list = Array.isArray(res?.data?.categories)
          ? res.data.categories
          : Array.isArray(res?.data)
          ? res.data
          : Array.isArray(res)
          ? res
          : [];
        setCategories(list);
      })
      .catch((err) => {
        if (cancelled) return;
        console.error("Failed to load categories:", err);
        setCategories([]);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  if (!isOpen || !event) return null;

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === "checkbox" ? checked : value,
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!formData.title.trim()) {
      setError("Event title is required");
      return;
    }

    try {
      setSubmitting(true);
      setError("");

      const payload = {
        title: formData.title.trim(),
        description: formData.description.trim(),
        banner_url: formData.banner_url.trim(),
        language: formData.language,
        age_restriction: parseInt(formData.age_restriction, 10) || 0,
        highlights: formData.highlights.trim(),
        rules: formData.rules.trim(),
        attendee_information: formData.attendee_information.trim(),
        contact_name: formData.contact_name.trim(),
        contact_phone: formData.contact_phone.trim(),
        contact_email: formData.contact_email.trim(),
      };

      // Only include structural fields for DRAFT events
      if (isDraft) {
        if (formData.category_id) {
          payload.category_id = formData.category_id;
        }
        if (formData.event_type) {
          payload.event_type = formData.event_type;
        }
        if (formData.online_url) {
          payload.online_url = formData.online_url;
        }
        if (formData.event_date) {
          const dateObj = new Date(formData.event_date);
          payload.event_date = dateObj.toISOString();
        }
        payload.is_all_day = formData.is_all_day;
        if (formData.booking_limit_per_user) {
          payload.booking_limit_per_user = parseInt(formData.booking_limit_per_user, 10);
        }
        payload.cancellation_allowed = formData.cancellation_allowed;
        payload.cancellation_deadline_hours = parseInt(formData.cancellation_deadline_hours, 10);
        payload.refund_policy = formData.refund_policy;
        payload.refund_percentage = parseInt(formData.refund_percentage, 10);
      }

      const res = await updateAdminEvent(event.id, payload);
      notify.success("Event updated successfully");
      if (onSuccess) {
        onSuccess(res?.data || res);
      }
      onClose();
    } catch (err) {
      console.error("Failed to update event:", err);
      const msg = err.response?.data?.message || err.message || "Failed to update event.";
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-[2px] animate-in fade-in duration-200">
      <div className="bg-white rounded-2xl shadow-2xl border border-[#dce2f7] w-full max-w-2xl max-h-[90vh] flex flex-col overflow-hidden">
        {/* Modal Header */}
        <div className="p-5 border-b border-[#dce2f7] flex items-center justify-between bg-[#f9f9ff] shrink-0">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-base font-bold text-[#141b2b]">
                Edit Event
              </h2>
              <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                isPublished
                  ? "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]"
                  : "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]"
              }`}>
                {event.status}
              </span>
            </div>
            <p className="text-xs text-[#3d4947]">
              {isPublished
                ? "Published event: descriptive metadata only (structural fields locked to preserve customer bookings)."
                : "Draft event: full editing mode."}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center text-[#3d4947] hover:bg-[#eef0fb] transition-colors cursor-pointer shrink-0"
          >
            ✕
          </button>
        </div>

        {/* Modal Body */}
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto p-5 space-y-4">
          {error && (
            <div className="rounded-lg bg-[#fff5f4] border border-[#ffdad6] text-[#ba1a1a] p-3 text-xs font-medium">
              {error}
            </div>
          )}

          {/* Descriptive Metadata Fields (Always Editable) */}
          <div className="space-y-3">
            <div>
              <label className="block text-xs font-bold text-[#141b2b] mb-1">
                Event Title *
              </label>
              <input
                type="text"
                name="title"
                value={formData.title}
                onChange={handleChange}
                required
                className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                placeholder="Enter event title"
              />
            </div>

            <div>
              <label className="block text-xs font-bold text-[#141b2b] mb-1">
                Description
              </label>
              <textarea
                name="description"
                rows="4"
                value={formData.description}
                onChange={handleChange}
                className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                placeholder="Detailed event description"
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-bold text-[#141b2b] mb-1">
                  Language
                </label>
                <input
                  type="text"
                  name="language"
                  value={formData.language}
                  onChange={handleChange}
                  className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-[#141b2b] mb-1">
                  Age Restriction (years)
                </label>
                <input
                  type="number"
                  name="age_restriction"
                  min="0"
                  max="100"
                  value={formData.age_restriction}
                  onChange={handleChange}
                  className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-[#141b2b] mb-1">
                Banner URL
              </label>
              <input
                type="text"
                name="banner_url"
                value={formData.banner_url}
                onChange={handleChange}
                className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                placeholder="S3 object key or URL"
              />
            </div>

            <div>
              <label className="block text-xs font-bold text-[#141b2b] mb-1">
                Highlights (one per line)
              </label>
              <textarea
                name="highlights"
                rows="2"
                value={formData.highlights}
                onChange={handleChange}
                className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
              />
            </div>

            <div>
              <label className="block text-xs font-bold text-[#141b2b] mb-1">
                Rules & Guidelines
              </label>
              <textarea
                name="rules"
                rows="2"
                value={formData.rules}
                onChange={handleChange}
                className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
              />
            </div>

            {/* Contact Information */}
            <div className="pt-2 border-t border-[#dce2f7]">
              <span className="text-xs font-bold text-[#141b2b] block mb-2">
                Event Contact Details
              </span>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                <div>
                  <label className="block text-[11px] text-[#3d4947] mb-0.5">Name</label>
                  <input
                    type="text"
                    name="contact_name"
                    value={formData.contact_name}
                    onChange={handleChange}
                    className="w-full text-xs px-2.5 py-1.5 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                  />
                </div>
                <div>
                  <label className="block text-[11px] text-[#3d4947] mb-0.5">Email</label>
                  <input
                    type="email"
                    name="contact_email"
                    value={formData.contact_email}
                    onChange={handleChange}
                    className="w-full text-xs px-2.5 py-1.5 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                  />
                </div>
                <div>
                  <label className="block text-[11px] text-[#3d4947] mb-0.5">Phone</label>
                  <input
                    type="text"
                    name="contact_phone"
                    value={formData.contact_phone}
                    onChange={handleChange}
                    className="w-full text-xs px-2.5 py-1.5 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* Structural Fields: Enabled ONLY for DRAFT */}
          {isDraft ? (
            <div className="pt-3 border-t border-[#dce2f7] space-y-3">
              <span className="text-xs font-bold text-[#00685f] block uppercase tracking-wider">
                Draft Structural Settings
              </span>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-bold text-[#141b2b] mb-1">
                    Category
                  </label>
                  <select
                    name="category_id"
                    value={formData.category_id}
                    onChange={handleChange}
                    className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b] bg-white"
                  >
                    <option value="">Select Category</option>
                    {(Array.isArray(categories) ? categories : []).map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-bold text-[#141b2b] mb-1">
                    Event Type
                  </label>
                  <select
                    name="event_type"
                    value={formData.event_type}
                    onChange={handleChange}
                    className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b] bg-white"
                  >
                    <option value="PHYSICAL">Physical Venue</option>
                    <option value="ONLINE">Online Stream</option>
                    <option value="HYBRID">Hybrid</option>
                  </select>
                </div>
              </div>

              {formData.event_type !== "PHYSICAL" && (
                <div>
                  <label className="block text-xs font-bold text-[#141b2b] mb-1">
                    Online Streaming URL
                  </label>
                  <input
                    type="url"
                    name="online_url"
                    value={formData.online_url}
                    onChange={handleChange}
                    className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                    placeholder="https://..."
                  />
                </div>
              )}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-bold text-[#141b2b] mb-1">
                    Event Date
                  </label>
                  <input
                    type="date"
                    name="event_date"
                    value={formData.event_date}
                    onChange={handleChange}
                    className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-[#141b2b] mb-1">
                    Max Tickets / User
                  </label>
                  <input
                    type="number"
                    name="booking_limit_per_user"
                    min="1"
                    max="50"
                    value={formData.booking_limit_per_user}
                    onChange={handleChange}
                    className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b]"
                  />
                </div>
              </div>
            </div>
          ) : (
            <div className="pt-3 border-t border-[#dce2f7] bg-[#f9f9ff] p-3 rounded-xl border border-dashed text-xs text-[#3d4947] flex items-center gap-2">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
              <span>Structural settings (Dates, Venue, Ticket Inventory) are locked on published events.</span>
            </div>
          )}

          {/* Modal Footer */}
          <div className="pt-4 border-t border-[#dce2f7] flex items-center justify-end gap-2 shrink-0">
            <button
              type="button"
              onClick={onClose}
              disabled={submitting}
              className="px-4 py-2 text-xs font-semibold text-[#3d4947] hover:bg-[#eef0fb] rounded-lg transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="bg-[#141b2b] hover:bg-[#2a3449] text-white text-xs font-semibold px-5 py-2 rounded-lg transition-colors cursor-pointer flex items-center gap-2 shadow-sm disabled:opacity-50"
            >
              {submitting ? "Saving Changes..." : "Save Changes"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
