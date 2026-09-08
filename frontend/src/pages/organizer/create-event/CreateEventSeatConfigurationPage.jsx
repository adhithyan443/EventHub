import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import PayoutBankAccountCard from "../../../components/organizer/PayoutBankAccountCard";
import useEventCreationStore from "../../../store/eventCreationStore";
import {
  ORGANIZER_ROUTES,
  TICKET_MODES,
  isOnlineEvent,
} from "../../../constants/eventConstants";

const COLOR_PALETTE = [
  { label: "Teal", value: "#00685f" },
  { label: "Indigo", value: "#4648d4" },
  { label: "Slate", value: "#6d7a77" },
  { label: "Amber", value: "#d97706" },
  { label: "Purple", value: "#9333ea" },
  { label: "Rose", value: "#e11d48" },
];

export default function CreateEventSeatConfigurationPage() {
  const navigate = useNavigate();

  const locationType = useEventCreationStore(
    (state) => state.locationType || state.eventType
  );
  const ticketMode = useEventCreationStore((state) => state.ticketMode);

  // Route guard: Online events cannot access seat configuration.
  // Direct URL access or invalid ticket mode is redirected to Ticket Types.
  useEffect(() => {
    if (isOnlineEvent(locationType) || ticketMode !== TICKET_MODES.SEATED) {
      navigate(ORGANIZER_ROUTES.CREATE_TICKET_TYPES, { replace: true });
    }
  }, [locationType, ticketMode, navigate]);

  const seatingConfig = useEventCreationStore((state) => state.seatingConfiguration);
  const toggleSeatSelection = useEventCreationStore((state) => state.toggleSeatSelection);
  const setSelectedSeatsStatus = useEventCreationStore((state) => state.setSelectedSeatsStatus);
  const assignCategoryToSelectedSeats = useEventCreationStore(
    (state) => state.assignCategoryToSelectedSeats
  );
  const generateLayout = useEventCreationStore((state) => state.generateLayout);
  const addCategory = useEventCreationStore((state) => state.addCategory);
  const removeCategory = useEventCreationStore((state) => state.removeCategory);
  const updateCategory = useEventCreationStore((state) => state.updateCategory);
  const addRowToCategory = useEventCreationStore((state) => state.addRowToCategory);
  const removeRowFromCategory = useEventCreationStore((state) => state.removeRowFromCategory);

  // Local state for Layout Settings inputs & Canvas zoom
  const [seatsPerRowInput, setSeatsPerRowInput] = useState(seatingConfig.seatsPerRow || 8);
  const [zoomLevel, setZoomLevel] = useState(100);

  // Modal states
  const [showAddModal, setShowAddModal] = useState(false);
  const [addForm, setAddForm] = useState({
    name: "",
    price: "",
    rowsCount: 1,
    color: "#00685f",
  });
  const [addError, setAddError] = useState("");

  const [showEditModal, setShowEditModal] = useState(false);
  const [editForm, setEditForm] = useState({
    id: "",
    name: "",
    price: "",
    color: "#00685f",
  });
  const [editError, setEditError] = useState("");

  const [categoryToRemove, setCategoryToRemove] = useState(null);

  const selectedSeatIds = seatingConfig.selectedSeatIds || [];
  const categories = seatingConfig.categories || [];
  const allRows = seatingConfig.rows || [];

  // Total capacity count
  const totalSeats =
    seatingConfig.totalCapacity ||
    categories.reduce(
      (acc, cat) =>
        acc + (cat.rows?.reduce((rAcc, r) => rAcc + (r.seats?.length || 0), 0) || 0),
      0
    );

  const handleBack = () => {
    navigate(ORGANIZER_ROUTES.CREATE_STEP_3);
  };

  const handleContinue = () => {
    navigate(ORGANIZER_ROUTES.CREATE_REVIEW);
  };

  const handleGenerateLayout = () => {
    const validSeats = Math.max(1, Math.min(30, Number(seatsPerRowInput) || 8));
    setSeatsPerRowInput(validSeats);
    generateLayout(validSeats);
  };

  // Add Category handler
  const handleOpenAddModal = () => {
    setAddForm({
      name: "",
      price: "",
      rowsCount: 1,
      color: COLOR_PALETTE[categories.length % COLOR_PALETTE.length].value,
    });
    setAddError("");
    setShowAddModal(true);
  };

  const handleSaveAddCategory = (e) => {
    e.preventDefault();
    if (!addForm.name.trim()) {
      setAddError("Category name is required.");
      return;
    }
    const exists = categories.some(
      (c) => c.name.trim().toLowerCase() === addForm.name.trim().toLowerCase()
    );
    if (exists) {
      setAddError("A category with this name already exists.");
      return;
    }
    if (!addForm.price.trim()) {
      setAddError("Price per seat is required.");
      return;
    }
    const validRows = Math.max(1, Math.min(20, Number(addForm.rowsCount) || 1));

    addCategory({
      name: addForm.name.trim(),
      tier: addForm.name.trim(),
      price: addForm.price.trim(),
      color: addForm.color,
      rowsCount: validRows,
    });

    setShowAddModal(false);
  };

  // Edit Category handler
  const handleOpenEditModal = (cat) => {
    setEditForm({
      id: cat.id,
      name: cat.name || "",
      price: cat.price || "",
      color: cat.color || "#00685f",
    });
    setEditError("");
    setShowEditModal(true);
  };

  const handleSaveEditCategory = (e) => {
    e.preventDefault();
    if (!editForm.name.trim()) {
      setEditError("Category name is required.");
      return;
    }
    const exists = categories.some(
      (c) =>
        c.id !== editForm.id &&
        c.name.trim().toLowerCase() === editForm.name.trim().toLowerCase()
    );
    if (exists) {
      setEditError("Another category with this name already exists.");
      return;
    }
    if (!editForm.price.trim()) {
      setEditError("Price per seat is required.");
      return;
    }

    const formattedPrice = editForm.price.trim().startsWith("₹")
      ? editForm.price.trim()
      : `₹${editForm.price.trim()}`;

    updateCategory(editForm.id, {
      name: editForm.name.trim(),
      tier: editForm.name.trim(),
      price: formattedPrice,
      color: editForm.color,
    });

    setShowEditModal(false);
  };

  // Remove Category handler
  const handleConfirmRemoveCategory = () => {
    if (categoryToRemove && categories.length > 1) {
      removeCategory(categoryToRemove.id);
      setCategoryToRemove(null);
    }
  };

  if (isOnlineEvent(locationType) || ticketMode !== TICKET_MODES.SEATED) {
    return null;
  }

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-8 max-w-6xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={4} />

        {/* Card Container */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          {/* Header */}
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff] flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="flex flex-col">
              <h1 className="text-xl font-bold text-[#141b2b]">Seat Configuration & Layout</h1>
              <p className="text-xs text-[#565e74] mt-1">
                Configure stage location, seating tiers, pricing per section, and visual seating map.
              </p>
            </div>

            <div className="flex items-center gap-3 bg-white border border-[#bcc9c6]/60 rounded-xl px-4 py-2 shadow-xs">
              <span className="text-xs text-[#565e74]">Total Venue Capacity:</span>
              <strong className="text-base font-bold text-[#00685f]">{totalSeats} Seats</strong>
            </div>
          </div>

          {/* Seating Builder Grid (Figma: 12-col grid) */}
          <div className="p-6 grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
            {/* Left Column: Visual Editor (8 Cols) */}
            <div className="lg:col-span-8 bg-[#f9f9ff] border border-[#dce2f7] rounded-xl shadow-xs overflow-hidden flex flex-col">
              {/* Editor Toolbar / Legend (Figma: Editor Toolbar / Legend) */}
              <div className="border-b border-[#dce2f7] bg-[#f9f9ff] px-4 py-3 flex items-center justify-between flex-wrap gap-3">
                {/* Legend items */}
                <div className="flex items-center gap-4 text-xs font-medium text-[#3d4947]">
                  {/* Available */}
                  <div className="flex items-center gap-1.5">
                    <div className="size-4 rounded-full border border-[#6d7a77] bg-[#f9f9ff]" />
                    <span>Available</span>
                  </div>

                  {/* Reserved */}
                  <div className="flex items-center gap-1.5">
                    <div className="size-4 rounded-full bg-[#00685f]" />
                    <span>Reserved</span>
                  </div>

                  {/* Blocked */}
                  <div className="flex items-center gap-1.5">
                    <div className="size-4 rounded-full bg-[#dce2f7] border border-[#bcc9c6] flex items-center justify-center">
                      <div className="size-1.5 bg-[#6d7a77] rounded-full" />
                    </div>
                    <span>Blocked</span>
                  </div>

                  {/* Selected */}
                  <div className="flex items-center gap-1.5">
                    <div className="size-4 rounded-full bg-[rgba(0,104,95,0.1)] border-2 border-[#00685f] flex items-center justify-center">
                      <svg className="size-2 text-[#00685f]" fill="currentColor" viewBox="0 0 16 16">
                        <path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z" />
                      </svg>
                    </div>
                    <span>Selected</span>
                  </div>
                </div>

                {/* Toolbar Zoom Controls */}
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setZoomLevel((z) => Math.max(70, z - 10))}
                    className="p-1.5 rounded hover:bg-gray-200 text-[#3d4947] text-xs font-bold transition-colors cursor-pointer"
                    title="Zoom Out"
                  >
                    🔍 -
                  </button>
                  <span className="text-xs text-[#565e74] font-medium">{zoomLevel}%</span>
                  <button
                    type="button"
                    onClick={() => setZoomLevel((z) => Math.min(130, z + 10))}
                    className="p-1.5 rounded hover:bg-gray-200 text-[#3d4947] text-xs font-bold transition-colors cursor-pointer"
                    title="Zoom In"
                  >
                    🔍 +
                  </button>
                </div>
              </div>

              {/* Editor Canvas (Figma: Stage + Sections/Tiers + Rows + Individual Seats) */}
              <div className="p-8 flex flex-col items-center min-h-[520px] overflow-auto">
                <div
                  className="flex flex-col items-center transition-transform origin-top w-full"
                  style={{ transform: `scale(${zoomLevel / 100})` }}
                >
                  {/* STAGE Podium (Figma: Stage) */}
                  <div className="h-[48px] max-w-[448px] w-full bg-[#dce2f7] border border-[#bcc9c6] rounded-b-[12px] flex items-center justify-center tracking-[2.8px] font-semibold text-[#3d4947] text-[14px] shadow-[inset_0px_2px_4px_0px_rgba(0,0,0,0.05)] mb-8 select-none">
                    STAGE
                  </div>

                  {/* Sections Hierarchy: Stage -> Category / Tier -> Rows -> Seats */}
                  <div className="flex flex-col gap-6 items-center w-full max-w-xl">
                    {categories.length > 0 ? (
                      categories.map((section) => (
                        <div
                          key={section.id}
                          className="w-full bg-white border border-[#dce2f7] rounded-[12px] p-4 shadow-2xs flex flex-col items-center gap-3 transition-all hover:border-[#bcc9c6]"
                        >
                          {/* Category / Tier Header */}
                          <div className="flex items-center justify-between w-full border-b border-[#dce2f7] pb-2.5 px-2 flex-wrap gap-2">
                            <div className="flex items-center gap-2">
                              <span
                                className="size-3.5 rounded-full shrink-0 shadow-xs"
                                style={{ backgroundColor: section.color || "#00685f" }}
                              />
                              <span className="font-bold text-[14px] text-[#141b2b] uppercase tracking-wider">
                                {section.name || section.tier}
                              </span>
                              <span className="text-[11px] font-semibold px-2 py-0.5 rounded-full bg-[#f1f3ff] text-[#565e74]">
                                {section.tier || section.name} Tier
                              </span>
                            </div>

                            <div className="flex items-center gap-4">
                              {section.price && (
                                <div className="flex items-center gap-1">
                                  <span className="font-bold text-[13px] text-[#00685f]">
                                    {section.price}
                                  </span>
                                  <span className="text-[11px] text-[#565e74]">/ seat</span>
                                </div>
                              )}

                              {/* Row Stepper Controls for this Category */}
                              <div className="flex items-center bg-[#f1f3ff] border border-[#dce2f7] rounded-lg p-0.5">
                                <button
                                  type="button"
                                  onClick={() => removeRowFromCategory(section.id)}
                                  disabled={section.rows?.length <= 1}
                                  className="size-6 flex items-center justify-center rounded text-xs font-bold text-[#3d4947] hover:bg-white hover:shadow-xs transition-all disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
                                  title="Decrease rows"
                                >
                                  −
                                </button>
                                <span className="text-[11px] font-semibold px-2 text-[#141b2b] min-w-[50px] text-center select-none">
                                  {section.rows?.length || 0} {section.rows?.length === 1 ? "Row" : "Rows"}
                                </span>
                                <button
                                  type="button"
                                  onClick={() => addRowToCategory(section.id)}
                                  className="size-6 flex items-center justify-center rounded text-xs font-bold text-[#3d4947] hover:bg-white hover:shadow-xs transition-all cursor-pointer"
                                  title="Add row to category"
                                >
                                  +
                                </button>
                              </div>
                            </div>
                          </div>

                          {/* Rows in this Category */}
                          <div className="flex flex-col gap-[10px] items-center w-full py-1">
                            {section.rows?.map((row) => (
                              <div
                                key={row.id || row.rowLetter}
                                className="flex gap-[8px] items-center justify-center w-full group/row"
                              >
                                {/* Row Indicator Label */}
                                <div className="w-[24px] text-center font-semibold text-[#3d4947] text-[14px] select-none shrink-0">
                                  {row.rowLetter}
                                </div>

                                {/* Individual Seat Circles (32px) */}
                                <div className="flex gap-[8px] items-center">
                                  {row.seats?.map((seat) => {
                                    const isSelected = selectedSeatIds.includes(seat.id);
                                    const seatCat =
                                      categories.find((c) => c.id === seat.categoryId) || section;

                                    let seatClasses =
                                      "size-[32px] rounded-full flex items-center justify-center text-[10px] cursor-pointer transition-all select-none ";
                                    let seatStyle = {};

                                    if (isSelected) {
                                      seatClasses +=
                                        "bg-[rgba(0,104,95,0.1)] border-2 border-[#00685f] text-[#00685f] font-bold shadow-xs scale-105";
                                    } else if (seat.status === "reserved") {
                                      seatClasses += "text-white font-medium shadow-xs";
                                      seatStyle = {
                                        backgroundColor: seatCat.color || section.color || "#00685f",
                                      };
                                    } else if (seat.status === "blocked") {
                                      seatClasses +=
                                        "bg-[#dce2f7] border border-[#bcc9c6] text-[#6d7a77]";
                                    } else {
                                      // Available
                                      seatClasses +=
                                        "bg-white border text-[#3d4947] hover:border-[#00685f] hover:text-[#00685f]";
                                      if (seat.categoryId && seat.categoryId !== section.id) {
                                        seatStyle = {
                                          borderColor: seatCat.color || "#00685f",
                                          boxShadow: `0 0 0 1px ${seatCat.color || "#00685f"}33`,
                                        };
                                      } else {
                                        seatClasses += " border-[#6d7a77]";
                                      }
                                    }

                                    return (
                                      <button
                                        key={seat.id}
                                        type="button"
                                        onClick={() => toggleSeatSelection(seat.id)}
                                        className={seatClasses}
                                        style={seatStyle}
                                        title={`Seat ${seat.id} (${seatCat.name} - ${seat.status})`}
                                      >
                                        {seat.status === "blocked" && !isSelected ? (
                                          <svg className="size-2.5 text-[#6d7a77]" fill="currentColor" viewBox="0 0 16 16">
                                            <path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z" />
                                            <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z" />
                                          </svg>
                                        ) : (
                                          seat.number
                                        )}
                                      </button>
                                    );
                                  })}
                                </div>

                                {/* Per-row Delete Action (enabled if category has > 1 row) */}
                                {section.rows.length > 1 && (
                                  <button
                                    type="button"
                                    onClick={() => removeRowFromCategory(section.id, row.id)}
                                    className="size-6 text-gray-300 hover:text-red-600 hover:bg-red-50 rounded flex items-center justify-center transition-colors cursor-pointer shrink-0 ml-1 opacity-0 group-hover/row:opacity-100"
                                    title={`Delete Row ${row.rowLetter}`}
                                  >
                                    <svg className="size-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                                      <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                    </svg>
                                  </button>
                                )}
                              </div>
                            ))}
                          </div>

                          {/* Quick Add Row Button */}
                          <div className="w-full flex justify-center pt-1">
                            <button
                              type="button"
                              onClick={() => addRowToCategory(section.id)}
                              className="text-xs font-medium text-[#00685f] hover:text-[#005049] hover:bg-[#00685f]/5 px-3 py-1 rounded-md transition-colors flex items-center gap-1.5 cursor-pointer border border-dashed border-[#00685f]/30"
                            >
                              <svg className="size-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                              </svg>
                              Add Row to {section.name}
                            </button>
                          </div>
                        </div>
                      ))
                    ) : (
                      // Fallback
                      <div className="text-sm text-[#565e74] py-8">
                        No categories found. Click "Add Category" to create one.
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* Right Column: Settings (4 Cols) */}
            <div className="lg:col-span-4 flex flex-col gap-5">
              {/* 1. Layout Settings Card (Figma: Layout Settings) */}
              <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-[17px] shadow-xs flex flex-col gap-4">
                <h3 className="font-semibold text-[14px] text-[#141b2b]">
                  Layout Settings
                </h3>

                <div className="flex gap-4 items-center justify-between">
                  <div className="flex flex-col gap-1 w-full">
                    <div className="flex items-center justify-between">
                      <label className="text-[12px] font-medium tracking-[0.6px] text-[#3d4947]">
                        Total Rows
                      </label>
                      <span className="text-[10px] text-[#565e74] bg-[#f1f3ff] px-1.5 py-0.5 rounded">
                        Per-tier
                      </span>
                    </div>
                    <input
                      type="text"
                      readOnly
                      disabled
                      value={`${allRows.length} (${categories.length} tiers)`}
                      className="w-full bg-gray-100/80 border border-[#bcc9c6]/80 rounded-[6px] p-2 text-sm text-[#565e74] cursor-not-allowed font-medium"
                    />
                  </div>

                  <div className="flex flex-col gap-1 w-full">
                    <label className="text-[12px] font-medium tracking-[0.6px] text-[#3d4947]">
                      Seats / Row
                    </label>
                    <input
                      type="number"
                      min="1"
                      max="30"
                      value={seatsPerRowInput}
                      onChange={(e) => setSeatsPerRowInput(e.target.value)}
                      className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-[6px] p-2 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                    />
                  </div>
                </div>

                <button
                  type="button"
                  onClick={handleGenerateLayout}
                  className="w-full bg-[#dce2f7] border border-[#bcc9c6] rounded-[8px] py-[9px] font-semibold text-[#141b2b] text-[14px] hover:bg-[#cfd6f0] transition-colors cursor-pointer"
                >
                  Generate Layout
                </button>
              </div>

              {/* 2. Selected Seats Card (Figma: Selected Seat Actions) */}
              <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-[17px] shadow-xs flex flex-col gap-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-[14px] text-[#141b2b]">
                    Selected Seats
                  </h3>
                  <span className="bg-[#008378] text-[#f4fffc] text-[12px] font-bold px-2 py-0.5 rounded-[4px]">
                    {selectedSeatIds.length}
                  </span>
                </div>

                <div className="bg-[#f1f3ff] border border-[#dce2f7] rounded-[4px] p-[9px] min-h-[40px] flex items-center">
                  <span className="text-[14px] text-[#3d4947] font-medium">
                    {selectedSeatIds.length > 0
                      ? selectedSeatIds.join(", ")
                      : "Click seats on canvas to select"}
                  </span>
                </div>

                <div className="flex flex-col gap-2">
                  <button
                    type="button"
                    disabled={selectedSeatIds.length === 0}
                    onClick={() => setSelectedSeatsStatus("available")}
                    className="w-full border border-[#bcc9c6] rounded-[8px] py-[9px] font-semibold text-[#141b2b] text-[14px] flex items-center justify-center gap-2 hover:bg-gray-50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <svg className="size-4 text-[#141b2b]" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                    Set Available
                  </button>

                  <button
                    type="button"
                    disabled={selectedSeatIds.length === 0}
                    onClick={() => setSelectedSeatsStatus("blocked")}
                    className="w-full border border-[rgba(186,26,26,0.5)] rounded-[8px] py-[9px] font-semibold text-[#ba1a1a] text-[14px] flex items-center justify-center gap-2 hover:bg-red-50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <svg className="size-4 text-[#ba1a1a]" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
                    </svg>
                    Block Selected
                  </button>
                </div>
              </div>

              {/* 3. Assign Category Card (Figma: Seat Category Mapping) */}
              <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-[17px] shadow-xs flex flex-col gap-3">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-[14px] text-[#141b2b]">
                    Assign Category
                  </h3>
                  <span className="text-[11px] text-[#565e74]">
                    {categories.length} {categories.length === 1 ? "Tier" : "Tiers"}
                  </span>
                </div>

                {selectedSeatIds.length > 0 && (
                  <p className="text-[11px] text-[#00685f] font-medium bg-[#00685f]/10 px-2 py-1 rounded">
                    Click a tier to assign to {selectedSeatIds.length} selected seat{selectedSeatIds.length > 1 ? "s" : ""}.
                  </p>
                )}

                <div className="flex flex-col gap-2">
                  {categories.map((cat) => (
                    <div
                      key={cat.id}
                      onClick={() => {
                        if (selectedSeatIds.length > 0) {
                          assignCategoryToSelectedSeats(cat.id);
                        }
                      }}
                      className={`border border-[#bcc9c6] rounded-[8px] p-[9px] flex items-center justify-between transition-all bg-white group ${
                        selectedSeatIds.length > 0
                          ? "cursor-pointer hover:bg-[#f0fdfa] hover:border-[#00685f]"
                          : "hover:border-gray-400"
                      }`}
                      title={
                        selectedSeatIds.length > 0
                          ? `Assign ${cat.name} to selected seats`
                          : `${cat.name} Tier (${cat.rows?.length || 0} rows)`
                      }
                    >
                      <div className="flex items-center gap-2.5">
                        <span
                          className="size-3 rounded-full shrink-0"
                          style={{ backgroundColor: cat.color || "#00685f" }}
                        />
                        <div className="flex flex-col">
                          <div className="flex items-center gap-1.5">
                            <span className="font-semibold text-[14px] text-[#141b2b]">
                              {cat.name}
                            </span>
                            <span className="text-[10px] text-[#565e74] bg-[#f1f3ff] px-1.5 py-0.5 rounded font-medium">
                              {cat.rows?.length || 0}r &bull; {(cat.rows?.length || 0) * (seatingConfig.seatsPerRow || 8)}s
                            </span>
                          </div>
                          <span className="text-[13px] font-medium text-[#3d4947]">
                            {cat.price}
                          </span>
                        </div>
                      </div>

                      <div className="flex items-center gap-1">
                        {/* Edit Category Button */}
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleOpenEditModal(cat);
                          }}
                          className="p-1 text-gray-400 hover:text-[#00685f] hover:bg-gray-100 rounded transition-colors cursor-pointer"
                          title="Edit category"
                        >
                          <svg className="size-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                          </svg>
                        </button>

                        {/* Remove Category Button (if > 1 category) */}
                        {categories.length > 1 && (
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              setCategoryToRemove(cat);
                            }}
                            className="p-1 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors cursor-pointer"
                            title="Remove category"
                          >
                            <svg className="size-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                              <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                          </button>
                        )}

                        {selectedSeatIds.length > 0 && (
                          <div className="size-5 rounded-full flex items-center justify-center text-[#00685f] ml-1">
                            <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2.5">
                              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                            </svg>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>

                {/* + Add Category Button */}
                <button
                  type="button"
                  onClick={handleOpenAddModal}
                  className="w-full mt-1 border border-dashed border-[#00685f] bg-[#00685f]/5 text-[#00685f] rounded-[8px] py-2.5 font-semibold text-[13px] flex items-center justify-center gap-1.5 hover:bg-[#00685f]/10 transition-colors cursor-pointer"
                >
                  <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2.5">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                  Add Category
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Payout Bank Account Card */}
        <PayoutBankAccountCard />
      </div>

      {/* Persistent Footer */}
      <EventCreationFooter
        onBack={handleBack}
        onContinue={handleContinue}
      />

      {/* Add Category Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4">
          <div className="bg-white rounded-2xl shadow-xl border border-gray-200 w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-150">
            <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
              <h3 className="font-bold text-base text-[#141b2b]">Add Seating Category</h3>
              <button
                type="button"
                onClick={() => setShowAddModal(false)}
                className="text-gray-400 hover:text-gray-600 p-1 rounded-md transition-colors"
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleSaveAddCategory} className="p-6 flex flex-col gap-4">
              {addError && (
                <div className="p-2.5 rounded-lg bg-red-50 border border-red-200 text-xs text-red-600 font-medium">
                  {addError}
                </div>
              )}

              <div className="flex flex-col gap-1">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Category Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  placeholder="e.g. Economy, Balcony, Front Row"
                  value={addForm.name}
                  onChange={(e) => setAddForm({ ...addForm, name: e.target.value })}
                  className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  autoFocus
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="flex flex-col gap-1">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Price per Seat <span className="text-red-500">*</span>
                  </label>
                  <div className="relative">
                    <span className="absolute left-3 top-2.5 text-xs text-gray-500">₹</span>
                    <input
                      type="text"
                      placeholder="499"
                      value={addForm.price.replace("₹", "")}
                      onChange={(e) => setAddForm({ ...addForm, price: e.target.value })}
                      className="w-full pl-7 pr-3 py-2 border border-[#bcc9c6] rounded-lg text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                    />
                  </div>
                </div>

                <div className="flex flex-col gap-1">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Number of Rows <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="15"
                    value={addForm.rowsCount}
                    onChange={(e) => setAddForm({ ...addForm, rowsCount: e.target.value })}
                    className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">Badge Color</label>
                <div className="flex items-center gap-2">
                  {COLOR_PALETTE.map((c) => (
                    <button
                      key={c.value}
                      type="button"
                      onClick={() => setAddForm({ ...addForm, color: c.value })}
                      className={`size-7 rounded-full transition-transform cursor-pointer flex items-center justify-center ${
                        addForm.color === c.value ? "ring-2 ring-offset-2 ring-[#00685f] scale-110" : "hover:scale-105"
                      }`}
                      style={{ backgroundColor: c.value }}
                      title={c.label}
                    >
                      {addForm.color === c.value && (
                        <svg className="size-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="3">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                    </button>
                  ))}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 mt-4 pt-3 border-t border-gray-100">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="px-4 py-2 rounded-lg text-sm font-semibold text-[#3d4947] hover:bg-gray-100 transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-5 py-2 rounded-lg text-sm font-semibold text-white bg-[#00685f] hover:bg-[#005049] transition-colors shadow-xs cursor-pointer"
                >
                  Add Category
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Category Modal */}
      {showEditModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4">
          <div className="bg-white rounded-2xl shadow-xl border border-gray-200 w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-150">
            <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
              <h3 className="font-bold text-base text-[#141b2b]">Edit Category</h3>
              <button
                type="button"
                onClick={() => setShowEditModal(false)}
                className="text-gray-400 hover:text-gray-600 p-1 rounded-md transition-colors"
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleSaveEditCategory} className="p-6 flex flex-col gap-4">
              {editError && (
                <div className="p-2.5 rounded-lg bg-red-50 border border-red-200 text-xs text-red-600 font-medium">
                  {editError}
                </div>
              )}

              <div className="flex flex-col gap-1">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Category Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={editForm.name}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                  className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  autoFocus
                />
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Price per Seat <span className="text-red-500">*</span>
                </label>
                <div className="relative">
                  <span className="absolute left-3 top-2.5 text-xs text-gray-500">₹</span>
                  <input
                    type="text"
                    value={editForm.price.replace("₹", "")}
                    onChange={(e) => setEditForm({ ...editForm, price: e.target.value })}
                    className="w-full pl-7 pr-3 py-2 border border-[#bcc9c6] rounded-lg text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
                  />
                </div>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">Badge Color</label>
                <div className="flex items-center gap-2">
                  {COLOR_PALETTE.map((c) => (
                    <button
                      key={c.value}
                      type="button"
                      onClick={() => setEditForm({ ...editForm, color: c.value })}
                      className={`size-7 rounded-full transition-transform cursor-pointer flex items-center justify-center ${
                        editForm.color === c.value ? "ring-2 ring-offset-2 ring-[#00685f] scale-110" : "hover:scale-105"
                      }`}
                      style={{ backgroundColor: c.value }}
                      title={c.label}
                    >
                      {editForm.color === c.value && (
                        <svg className="size-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="3">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                    </button>
                  ))}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 mt-4 pt-3 border-t border-gray-100">
                <button
                  type="button"
                  onClick={() => setShowEditModal(false)}
                  className="px-4 py-2 rounded-lg text-sm font-semibold text-[#3d4947] hover:bg-gray-100 transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-5 py-2 rounded-lg text-sm font-semibold text-white bg-[#00685f] hover:bg-[#005049] transition-colors shadow-xs cursor-pointer"
                >
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Remove Category Confirmation Modal */}
      {categoryToRemove && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4">
          <div className="bg-white rounded-2xl shadow-xl border border-gray-200 w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-150">
            <div className="p-6 flex flex-col gap-4">
              <div className="size-12 rounded-full bg-red-100 text-red-600 flex items-center justify-center mx-auto">
                <svg className="size-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>

              <div className="flex flex-col items-center text-center gap-1">
                <h3 className="text-lg font-bold text-[#141b2b]">
                  Remove {categoryToRemove.name} Category?
                </h3>
                <p className="text-xs text-[#565e74] max-w-xs mt-1 leading-relaxed">
                  This category currently contains{" "}
                  <strong>{categoryToRemove.rows?.length || 0} rows</strong> and{" "}
                  <strong>
                    {(categoryToRemove.rows?.length || 0) * (seatingConfig.seatsPerRow || 8)} seats
                  </strong>
                  . Removing it will remove these rows from the seating layout and re-sequence all remaining rows.
                </p>
              </div>

              <div className="flex items-center justify-center gap-3 mt-3 pt-3 border-t border-gray-100">
                <button
                  type="button"
                  onClick={() => setCategoryToRemove(null)}
                  className="px-4 py-2 rounded-lg text-sm font-semibold text-[#3d4947] hover:bg-gray-100 transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleConfirmRemoveCategory}
                  className="px-5 py-2 rounded-lg text-sm font-semibold text-white bg-[#ba1a1a] hover:bg-red-700 transition-colors shadow-xs cursor-pointer"
                >
                  Remove Category
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

