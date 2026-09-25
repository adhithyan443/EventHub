import { useState } from "react";
import { useNavigate } from "react-router-dom";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import PayoutBankAccountCard from "../../../components/organizer/PayoutBankAccountCard";
import useEventCreationStore from "../../../store/eventCreationStore";
import { ORGANIZER_ROUTES } from "../../../constants/eventConstants";

const MAX_TICKET_NAME_LENGTH = 100;
const MAX_TICKET_DESCRIPTION_LENGTH = 500;
const MAX_TICKET_QUANTITY = 1000000;
const MAX_TICKETS_PER_ORDER = 20;

const EMPTY_TICKET = {
  name: "",
  price: "",
  capacity: "",
  description: "",
};

const getTodayDateString = () => {
  const today = new Date();

  return [
    today.getFullYear(),
    String(today.getMonth() + 1).padStart(2, "0"),
    String(today.getDate()).padStart(2, "0"),
  ].join("-");
};

const isValidDateString = (value) => {
  if (!value) {
    return false;
  }

  const date = new Date(`${value}T00:00:00`);

  return !Number.isNaN(date.getTime());
};

export default function CreateEventTicketTypesPage() {
  const navigate = useNavigate();

  const ticketTypes = useEventCreationStore((state) => state.ticketTypes);

  const addTicketType = useEventCreationStore((state) => state.addTicketType);

  const updateTicketType = useEventCreationStore(
    (state) => state.updateTicketType
  );

  const removeTicketType = useEventCreationStore(
    (state) => state.removeTicketType
  );

  const ticketSalesSettings = useEventCreationStore(
    (state) => state.ticketSalesSettings
  );

  const updateTicketSalesSettings = useEventCreationStore(
    (state) => state.updateTicketSalesSettings
  );

  const dateTime = useEventCreationStore((state) => state.dateTime);

  const [showAddModal, setShowAddModal] = useState(false);
  const [editingTicketId, setEditingTicketId] = useState(null);

  const [newTicket, setNewTicket] = useState({ ...EMPTY_TICKET });

  const [validationError, setValidationError] = useState("");

  const handleBack = () => {
    navigate(ORGANIZER_ROUTES.CREATE_STEP_3);
  };

  /*
   * Validate all persisted ticket types.
   */
  const validateTicketTypes = () => {
    if (!ticketTypes || ticketTypes.length === 0) {
      return "Please add at least one ticket type.";
    }

    const ticketNames = new Set();

    for (let index = 0; index < ticketTypes.length; index += 1) {
      const ticket = ticketTypes[index];

      const ticketNumber = index + 1;

      const name = String(ticket.name || "").trim();
      const description = String(ticket.description || "").trim();

      /*
       * Ticket name
       */
      if (!name) {
        return `Ticket ${ticketNumber}: Ticket name is required.`;
      }

      if (name.length < 2) {
        return `Ticket ${ticketNumber}: Ticket name must be at least 2 characters.`;
      }

      if (name.length > MAX_TICKET_NAME_LENGTH) {
        return `Ticket ${ticketNumber}: Ticket name must not exceed ${MAX_TICKET_NAME_LENGTH} characters.`;
      }

      /*
       * Duplicate ticket names
       */
      const normalizedName = name.toLowerCase();

      if (ticketNames.has(normalizedName)) {
        return `Ticket ${ticketNumber}: A ticket type with the name "${name}" already exists.`;
      }

      ticketNames.add(normalizedName);

      /*
       * Price
       */
      if (
        ticket.price === "" ||
        ticket.price === null ||
        ticket.price === undefined
      ) {
        return `Ticket ${ticketNumber}: Ticket price is required.`;
      }

      const price = Number(ticket.price);

      if (!Number.isFinite(price) || price < 0) {
        return `Ticket ${ticketNumber}: Ticket price must be a valid amount greater than or equal to 0.`;
      }

      /*
       * Quantity
       */
      if (
        ticket.capacity === "" ||
        ticket.capacity === null ||
        ticket.capacity === undefined
      ) {
        return `Ticket ${ticketNumber}: Ticket quantity is required.`;
      }

      const capacity = Number(ticket.capacity);

      if (!Number.isInteger(capacity) || capacity <= 0) {
        return `Ticket ${ticketNumber}: Ticket quantity must be a positive whole number.`;
      }

      if (capacity > MAX_TICKET_QUANTITY) {
        return `Ticket ${ticketNumber}: Ticket quantity cannot exceed ${MAX_TICKET_QUANTITY}.`;
      }

      /*
       * Description
       */
      if (description.length > MAX_TICKET_DESCRIPTION_LENGTH) {
        return `Ticket ${ticketNumber}: Description must not exceed ${MAX_TICKET_DESCRIPTION_LENGTH} characters.`;
      }
    }

    return null;
  };

  /*
   * Validate ticket sales settings.
   */
  const validateTicketSalesSettings = () => {
    const salesStartDate = ticketSalesSettings?.salesStartDate || "";
    const salesEndDate = ticketSalesSettings?.salesEndDate || "";

    const maxTicketsPerBooking =
      ticketSalesSettings?.maxTicketsPerBooking ?? "";

    const eventDate = dateTime?.eventDate || "";

    /*
     * Event date
     */
    if (!eventDate) {
      return "Event date is missing. Please go back to Step 2 and select an event date.";
    }

    if (!isValidDateString(eventDate)) {
      return "The event date is invalid. Please go back to Step 2 and select a valid event date.";
    }

    /*
     * Sales start date
     */
    if (!salesStartDate) {
      return "Sales start date is required.";
    }

    if (!isValidDateString(salesStartDate)) {
      return "Sales start date must be a valid date.";
    }

    /*
     * Sales end date
     */
    if (!salesEndDate) {
      return "Sales end date is required.";
    }

    if (!isValidDateString(salesEndDate)) {
      return "Sales end date must be a valid date.";
    }

    /*
     * Sales date order
     */
    if (salesStartDate > salesEndDate) {
      return "Sales start date cannot be after the sales end date.";
    }

    /*
     * Sales cannot continue after the event.
     */
    if (salesEndDate > eventDate) {
      return "Sales end date cannot be after the event date.";
    }

    if (salesStartDate > eventDate) {
      return "Sales start date cannot be after the event date.";
    }

    /*
     * Sales cannot start in the past.
     */
    const today = getTodayDateString();

    if (salesStartDate < today) {
      return "Sales start date cannot be in the past.";
    }

    /*
     * Maximum tickets per order
     */
    if (
      maxTicketsPerBooking === "" ||
      maxTicketsPerBooking === null ||
      maxTicketsPerBooking === undefined
    ) {
      return "Maximum tickets per order is required.";
    }

    const maxTickets = Number(maxTicketsPerBooking);

    if (!Number.isInteger(maxTickets) || maxTickets <= 0) {
      return "Maximum tickets per order must be a positive whole number.";
    }

    if (maxTickets > MAX_TICKETS_PER_ORDER) {
      return `Maximum tickets per order cannot exceed ${MAX_TICKETS_PER_ORDER}.`;
    }

    /*
     * Maximum tickets per order cannot exceed
     * the total available ticket capacity.
     */
    const totalCapacity = ticketTypes.reduce(
      (total, ticket) => total + Number(ticket.capacity || 0),
      0
    );

    if (maxTickets > totalCapacity) {
      return "Maximum tickets per order cannot exceed the total available ticket quantity.";
    }

    return null;
  };

  /*
   * Validate the complete Step 4.
   */
  const handleContinue = () => {
    setValidationError("");

    const ticketTypesError = validateTicketTypes();

    if (ticketTypesError) {
      setValidationError(ticketTypesError);
      return;
    }

    const ticketSalesSettingsError = validateTicketSalesSettings();

    if (ticketSalesSettingsError) {
      setValidationError(ticketSalesSettingsError);
      return;
    }

    navigate(ORGANIZER_ROUTES.CREATE_REVIEW);
  };

  const openAddTicketModal = () => {
    setValidationError("");
    setEditingTicketId(null);
    setNewTicket({ ...EMPTY_TICKET });
    setShowAddModal(true);
  };

  const handleEditTicket = (ticket) => {
    setValidationError("");
    setEditingTicketId(ticket.id);

    setNewTicket({
      name: ticket.name || "",
      price: String(ticket.price ?? ""),
      capacity: String(ticket.capacity ?? ""),
      description: ticket.description || "",
    });

    setShowAddModal(true);
  };

  const closeTicketModal = () => {
    setShowAddModal(false);
    setEditingTicketId(null);
    setNewTicket({ ...EMPTY_TICKET });
  };

  /*
   * Validate and create/update a ticket.
   */
  const handleCreateTicket = (e) => {
    e.preventDefault();

    setValidationError("");

    const name = newTicket.name.trim();
    const description = newTicket.description.trim();

    /*
     * Ticket name validation
     */
    if (!name) {
      setValidationError("Ticket name is required.");
      return;
    }

    if (name.length < 2) {
      setValidationError("Ticket name must be at least 2 characters.");
      return;
    }

    if (name.length > MAX_TICKET_NAME_LENGTH) {
      setValidationError(
        `Ticket name must not exceed ${MAX_TICKET_NAME_LENGTH} characters.`
      );
      return;
    }

    /*
     * Price validation
     */
    if (newTicket.price === "") {
      setValidationError("Ticket price is required.");
      return;
    }

    const price = Number(newTicket.price);

    if (!Number.isFinite(price) || price < 0) {
      setValidationError(
        "Ticket price must be a valid amount greater than or equal to 0."
      );
      return;
    }

    /*
     * Quantity validation
     */
    if (newTicket.capacity === "") {
      setValidationError("Ticket quantity is required.");
      return;
    }

    const capacity = Number(newTicket.capacity);

    if (!Number.isInteger(capacity) || capacity <= 0) {
      setValidationError(
        "Ticket quantity must be a positive whole number."
      );
      return;
    }

    if (capacity > MAX_TICKET_QUANTITY) {
      setValidationError(
        `Ticket quantity cannot exceed ${MAX_TICKET_QUANTITY}.`
      );
      return;
    }

    /*
     * Description validation
     */
    if (description.length > MAX_TICKET_DESCRIPTION_LENGTH) {
      setValidationError(
        `Description must not exceed ${MAX_TICKET_DESCRIPTION_LENGTH} characters.`
      );
      return;
    }

    /*
     * Duplicate ticket name validation
     */
    const isDuplicate = ticketTypes.some(
      (ticket) =>
        ticket.id !== editingTicketId &&
        String(ticket.name || "").trim().toLowerCase() === name.toLowerCase()
    );

    if (isDuplicate) {
      setValidationError("A ticket type with this name already exists.");
      return;
    }

    const ticketData = {
      name,
      price,
      capacity,
      description,
    };

    if (editingTicketId) {
      updateTicketType(editingTicketId, ticketData);
    } else {
      addTicketType(ticketData);
    }

    closeTicketModal();
  };

  const handleDeleteTicket = (ticketId) => {
    removeTicketType(ticketId);
    setValidationError("");
  };

  const handleSalesSettingChange = (field, value) => {
    updateTicketSalesSettings({
      [field]: value,
    });

    setValidationError("");
  };

  const todayDate = getTodayDateString();

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-4 sm:p-6 lg:p-8 max-w-4xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={4} />

        {/* Validation Error */}
        {validationError && !showAddModal && (
          <div className="rounded-lg border border-[#ffdad6] bg-[#fff5f3] px-4 py-3">
            <p className="text-sm font-medium text-[#ba1a1a]">
              {validationError}
            </p>
          </div>
        )}

        {/* Card 1: Ticket Types */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          <div className="p-4 sm:p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff] flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div className="flex flex-col">
              <h1 className="text-xl font-bold text-[#141b2b]">
                Ticket Types
              </h1>

              <p className="text-xs text-[#565e74] mt-1">
                Create different ticket tiers, set quantities, and manage
                pricing for general admission.
              </p>
            </div>

            <button
              type="button"
              onClick={openAddTicketModal}
              className="flex items-center gap-2 bg-[#00685f] hover:bg-[#005550] text-white text-xs font-semibold px-4 py-2.5 rounded-lg shadow-sm transition-colors cursor-pointer self-start sm:self-auto shrink-0"
            >
              <svg
                className="size-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2.5"
                  d="M12 4v16m8-8H4"
                />
              </svg>

              <span>Add Ticket Type</span>
            </button>
          </div>

          <div className="p-4 sm:p-6 flex flex-col gap-4">
            {ticketTypes.length === 0 ? (
              <div className="p-8 text-center text-xs text-[#565e74] border border-dashed border-[#bcc9c6] rounded-xl">
                No ticket types created yet. Click &quot;Add Ticket Type&quot;
                above to create one.
              </div>
            ) : (
              ticketTypes.map((ticket) => (
                <div
                  key={ticket.id}
                  className="bg-[#f9f9ff] border border-[#bcc9c6]/60 rounded-xl p-5 flex flex-col md:flex-row md:items-center justify-between gap-4"
                >
                  <div className="flex items-start gap-3.5">
                    <div className="size-10 rounded-full bg-[#00685f]/15 flex items-center justify-center text-[#00685f] shrink-0 font-bold">
                      🎟️
                    </div>

                    <div className="flex flex-col">
                      <div className="flex items-center gap-2">
                        <h3 className="font-bold text-[14px] text-[#141b2b]">
                          {ticket.name}
                        </h3>

                        <span
                          className={`text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider ${ticket.status === "ACTIVE"
                              ? "bg-emerald-100 text-emerald-800"
                              : "bg-gray-200 text-gray-700"
                            }`}
                        >
                          {ticket.status || "ACTIVE"}
                        </span>
                      </div>

                      <p className="text-xs text-[#565e74] mt-0.5">
                        {ticket.description}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-6 self-end md:self-center">
                    <div className="flex flex-col text-right">
                      <span className="text-base font-bold text-[#141b2b]">
                        ${Number(ticket.price).toFixed(2)}
                      </span>

                      <span className="text-[11px] text-[#565e74]">
                        {ticket.capacity} tickets total
                      </span>
                    </div>

                    <div className="flex items-center gap-2">
                      {/* Edit */}
                      <button
                        type="button"
                        onClick={() => handleEditTicket(ticket)}
                        className="p-1.5 text-gray-400 hover:text-[#00685f] rounded transition-colors cursor-pointer"
                        title="Edit ticket"
                      >
                        <svg
                          className="size-4"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth="2"
                            d="M16.862 3.487a2.25 2.25 0 013.182 3.182L8.25 18.464 4 19.5l1.036-4.25L16.862 3.487z"
                          />
                        </svg>
                      </button>

                      {/* Delete */}
                      <button
                        type="button"
                        onClick={() => handleDeleteTicket(ticket.id)}
                        className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors cursor-pointer"
                        title="Remove ticket"
                      >
                        <svg
                          className="size-4"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth="2"
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a2 2 0 00-2-2h-3a2 2 0 00-2 2v1H4"
                          />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Card 2: Ticketing Sales Settings */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
            <h2 className="text-base font-bold text-[#141b2b]">
              Ticketing Settings
            </h2>

            <p className="text-xs text-[#565e74] mt-1">
              Sales windows, maximum tickets allowed per customer, and
              purchase rules.
            </p>
          </div>

          <div className="p-6 grid grid-cols-1 md:grid-cols-3 gap-5">
            {/* Sales Start Date */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Sales Start Date{" "}
                <span className="text-[#ba1a1a]">*</span>
              </label>

              <input
                type="date"
                required
                min={todayDate}
                max={dateTime?.eventDate || undefined}
                value={ticketSalesSettings.salesStartDate}
                onChange={(e) =>
                  handleSalesSettingChange(
                    "salesStartDate",
                    e.target.value
                  )
                }
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>

            {/* Sales End Date */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Sales End Date{" "}
                <span className="text-[#ba1a1a]">*</span>
              </label>

              <input
                type="date"
                required
                min={ticketSalesSettings.salesStartDate || todayDate}
                max={dateTime?.eventDate || undefined}
                value={ticketSalesSettings.salesEndDate}
                onChange={(e) =>
                  handleSalesSettingChange(
                    "salesEndDate",
                    e.target.value
                  )
                }
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>

            {/* Maximum Tickets */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Max Tickets Per Order{" "}
                <span className="text-[#ba1a1a]">*</span>
              </label>

              <input
                type="number"
                required
                min={1}
                max={MAX_TICKETS_PER_ORDER}
                step={1}
                value={ticketSalesSettings.maxTicketsPerBooking}
                onChange={(e) =>
                  handleSalesSettingChange(
                    "maxTicketsPerBooking",
                    e.target.value
                  )
                }
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>
          </div>
        </div>

        {/* Card 3: Payout Account */}
        <PayoutBankAccountCard />
      </div>

      {/* Add / Edit Ticket Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-[#bcc9c6] flex flex-col gap-5">
            <div className="flex items-center justify-between">
              <h3 className="font-bold text-lg text-[#141b2b]">
                {editingTicketId
                  ? "Edit Ticket Type"
                  : "Add New Ticket Type"}
              </h3>

              <button
                type="button"
                onClick={closeTicketModal}
                className="text-gray-400 hover:text-gray-700 text-lg"
                title="Close"
              >
                &times;
              </button>
            </div>

            {/* Modal Validation Error */}
            {validationError && (
              <div className="rounded-lg border border-[#ffdad6] bg-[#fff5f3] px-4 py-3">
                <p className="text-sm font-medium text-[#ba1a1a]">
                  {validationError}
                </p>
              </div>
            )}

            <form
              onSubmit={handleCreateTicket}
              className="flex flex-col gap-4"
            >
              {/* Ticket Name */}
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Ticket Name{" "}
                  <span className="text-[#ba1a1a]">*</span>
                </label>

                <input
                  type="text"
                  required
                  maxLength={MAX_TICKET_NAME_LENGTH}
                  placeholder="e.g. VIP Pass, Student Discount"
                  value={newTicket.name}
                  onChange={(e) => {
                    setNewTicket({
                      ...newTicket,
                      name: e.target.value,
                    });

                    setValidationError("");
                  }}
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                />
              </div>

              {/* Price and Quantity */}
              <div className="grid grid-cols-2 gap-4">
                {/* Price */}
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Price ($){" "}
                    <span className="text-[#ba1a1a]">*</span>
                  </label>

                  <input
                    type="number"
                    required
                    min={0}
                    step="0.01"
                    placeholder="99"
                    value={newTicket.price}
                    onChange={(e) => {
                      setNewTicket({
                        ...newTicket,
                        price: e.target.value,
                      });

                      setValidationError("");
                    }}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                {/* Quantity */}
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Quantity Available{" "}
                    <span className="text-[#ba1a1a]">*</span>
                  </label>

                  <input
                    type="number"
                    required
                    min={1}
                    max={MAX_TICKET_QUANTITY}
                    step={1}
                    placeholder="500"
                    value={newTicket.capacity}
                    onChange={(e) => {
                      setNewTicket({
                        ...newTicket,
                        capacity: e.target.value,
                      });

                      setValidationError("");
                    }}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                  />
                </div>
              </div>

              {/* Description */}
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Description
                </label>

                <textarea
                  rows={2}
                  maxLength={MAX_TICKET_DESCRIPTION_LENGTH}
                  placeholder="What is included with this ticket?"
                  value={newTicket.description}
                  onChange={(e) => {
                    setNewTicket({
                      ...newTicket,
                      description: e.target.value,
                    });

                    setValidationError("");
                  }}
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-3 text-sm focus:outline-none focus:border-[#00685f]"
                />

                <span className="text-[10px] text-[#565e74] text-right">
                  {newTicket.description.length}/
                  {MAX_TICKET_DESCRIPTION_LENGTH}
                </span>
              </div>

              {/* Actions */}
              <div className="flex items-center justify-end gap-3 mt-2">
                <button
                  type="button"
                  onClick={closeTicketModal}
                  className="px-4 py-2 rounded-lg border border-[#bcc9c6] text-xs font-semibold text-[#565e74] hover:bg-gray-50"
                >
                  Cancel
                </button>

                <button
                  type="submit"
                  className="px-5 py-2 rounded-lg bg-[#00685f] hover:bg-[#005550] text-white text-xs font-semibold shadow-sm"
                >
                  {editingTicketId ? "Save Changes" : "Create Ticket"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Persistent Footer */}
      <EventCreationFooter
        onBack={handleBack}
        onContinue={handleContinue}
      />
    </div>
  );
}