import { create } from "zustand";

import {
  TICKET_MODES,
  LOCATION_TYPES,
  isOnlineEvent,
} from "../constants/eventConstants";

const initialVenue = {
  google_place_id: "",
  name: "",
  address: "",
  city: "",
  state: "",
  country: "",
  postal_code: "",
  latitude: null,
  longitude: null,
};

const initialBasicInfo = {
  title: "",
  category: "",
  categoryId: "",
  description: "",
  banner: "",
  bannerFile: null,
  bannerPreviewUrl: "",
  language: "",
  ageRestriction: "",
};

const initialDateTime = {
  eventDate: "",
  startTime: "",
  endTime: "",
  isAllDay: false,
};

const initialEventDetails = {
  highlights: "",
  rules: "",
  contactInformation: {
    name: "",
    phone: "",
    email: "",
  },
  cancellationPolicy: {
    allowCancellation: true,
    cancellationDeadline: "48 hours before event",
    refundPolicy: "PARTIAL",
    refundPercentage: 80,
    organizerPolicyAccepted: false,
  },
  attendeeInformation: "",
  visibility: "PUBLIC",
};

const initialTicketTypes = [];

const initialTicketSalesSettings = {
  salesStartDate: "",
  salesEndDate: "",
  maxTicketsPerBooking: "",
};

// const initialPayoutAccount = {
//   bankName: "",
//   accountHolder: "",
//   accountNumberMasked: "",
// };

const initialSeatingConfig = {
  layoutName: "",
  categories: [],
  sections: [],
  rows: [],
  columns: [],
  columnLabelType: "ALPHA",
  selectedSeatIds: [],
  totalCapacity: 0,
};

const mapHoursToCancellationDeadline = (hours) => {
  if (hours === 24) return "24 hours before event";
  if (hours === 48) return "48 hours before event";
  if (hours === 168) return "7 days before event";
  if (hours === 336) return "14 days before event";
  if (hours > 0) return `${hours} hours before event`;
  return "48 hours before event";
};

const pruneSelectedSeatIds = (
  oldCategories,
  newCategories,
  selectedSeatIds
) => {
  if (!selectedSeatIds || selectedSeatIds.length === 0) {
    return [];
  }

  const oldSeatRowMap = new Map();

  for (const category of oldCategories || []) {
    for (const row of category.rows || []) {
      for (const seat of row.seats || []) {
        if (seat.id) {
          oldSeatRowMap.set(seat.id, row.id);
        }
      }
    }
  }

  const newSeatRowMap = new Map();

  for (const category of newCategories || []) {
    for (const row of category.rows || []) {
      for (const seat of row.seats || []) {
        if (seat.id) {
          newSeatRowMap.set(seat.id, row.id);
        }
      }
    }
  }

  return selectedSeatIds.filter((seatId) => {
    const oldRowId = oldSeatRowMap.get(seatId);
    const newRowId = newSeatRowMap.get(seatId);

    return oldRowId && newRowId && oldRowId === newRowId;
  });
};

const getAlphabeticLabel = (orderIndex) => {
  if (orderIndex < 0) return "A";
  let result = "";
  let curr = orderIndex + 1;
  while (curr > 0) {
    curr--;
    const rem = curr % 26;
    result = String.fromCharCode(65 + rem) + result;
    curr = Math.floor(curr / 26);
  }
  return result;
};

const generateDefaultColumns = (count, labelType = "ALPHA") => {
  const safeCount = Math.max(1, Number(count) || 8);
  return Array.from({ length: safeCount }, (_, idx) => ({
    id: `col-${Date.now()}-${idx}`,
    label: labelType === "NUMERIC" ? String(idx + 1) : getAlphabeticLabel(idx),
    order_index: idx,
  }));
};

const rebuildLayout = (categories, columns = [], columnLabelType = "ALPHA") => {
  let rowIndex = 0;

  let sortedColumns = [];
  if (Array.isArray(columns) && columns.length > 0) {
    sortedColumns = [...columns];
  } else if (typeof columns === "number") {
    sortedColumns = generateDefaultColumns(columns, columnLabelType);
  } else {
    sortedColumns = generateDefaultColumns(8, columnLabelType);
  }
  sortedColumns.sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0));

  return categories.map((category) => {
    const updatedRows = (category.rows || []).map((row) => {
      const rowLetter =
        rowIndex < 26
          ? String.fromCharCode(65 + rowIndex)
          : `R${rowIndex + 1}`;

      rowIndex++;

      const existingSeatByCol = new Map();
      (row.seats || []).forEach((seat, sIdx) => {
        if (seat.columnId) {
          existingSeatByCol.set(seat.columnId, seat);
        }
        if (seat.gridCol !== undefined) {
          existingSeatByCol.set(`grid-${seat.gridCol}`, seat);
        }
        existingSeatByCol.set(`idx-${sIdx}`, seat);
      });

      const seats = sortedColumns.map((col, colIdx) => {
        const existingSeat =
          existingSeatByCol.get(col.id) ||
          existingSeatByCol.get(`grid-${col.order_index ?? colIdx}`) ||
          existingSeatByCol.get(`idx-${colIdx}`);

        const number = colIdx + 1;
        const isGap = existingSeat?.status === "gap";

        return {
          ...existingSeat,
          id:
            existingSeat?.id && !existingSeat.id.startsWith("seat-")
              ? existingSeat.id
              : `${rowLetter}${number}`,
          number: existingSeat?.number || number,
          columnId: col.id,
          gridCol: col.order_index ?? colIdx,
          columnLabel: col.label,
          status: isGap ? "gap" : existingSeat?.status || "available",
          categoryId: existingSeat?.categoryId || category.id,
        };
      });

      return {
        ...row,
        id: row.id || `row-${rowLetter}-${category.id}`,
        name: `Row ${rowLetter}`,
        rowLetter,
        seats,
      };
    });

    return {
      ...category,
      rows: updatedRows,
    };
  });
};

const useEventCreationStore = create((set, get) => ({
  // Edit Mode
  isEditMode: false,
  editingEventId: null,

  // Location & Event Type
  locationType: LOCATION_TYPES.PHYSICAL,
  eventType: LOCATION_TYPES.PHYSICAL,
  onlineUrl: "",

  // Venue
  venue: { ...initialVenue },

  // Step 1: Basic Information
  basicInformation: { ...initialBasicInfo },

  // Step 2: Date & Time + Details
  dateTime: { ...initialDateTime },
  eventDetails: { ...initialEventDetails },

  // Step 3: Ticket Mode
  ticketMode: TICKET_MODES.GENERAL,

  // Step 4A: Ticket Types
  ticketTypes: initialTicketTypes,
  ticketSalesSettings: {
    ...initialTicketSalesSettings,
  },

  // Payout account
  // payoutAccount: { ...initialPayoutAccount },

  // Step 4B: Seat Configuration
  seatingConfiguration: {
    ...initialSeatingConfig,
  },

  // --------------------------------------------------
  // Location & Event Type Actions
  // --------------------------------------------------

  setLocationType: (type) =>
    set((state) => {
      const isSwitchingToOnline = isOnlineEvent(type);

      let nextVenue = state.venue;
      let nextTicketMode = state.ticketMode;
      let nextSeatingConfig = state.seatingConfiguration;

      if (isSwitchingToOnline) {
        nextVenue = { ...initialVenue };
        // In edit mode, never override hydrated ticketMode
        if (!state.isEditMode) {
          nextTicketMode = TICKET_MODES.GENERAL;
          nextSeatingConfig = { ...initialSeatingConfig };
        }
      }

      return {
        locationType: type,
        eventType: type,
        venue: nextVenue,
        ticketMode: nextTicketMode,
        seatingConfiguration: nextSeatingConfig,
      };
    }),

  setEventType: (type) => get().setLocationType(type),

  setOnlineUrl: (url) =>
    set({
      onlineUrl: url,
    }),

  setVenue: (venueData) =>
    set((state) => ({
      venue: {
        ...state.venue,
        ...venueData,
      },
    })),

  // --------------------------------------------------
  // Ticket Mode
  // --------------------------------------------------

  setTicketMode: (mode) =>
    set((state) => {
      // Immutability Guard: Never mutate mode in edit mode
      if (state.isEditMode) {
        return {};
      }

      // Online events cannot use reserved seating.
      if (
        isOnlineEvent(state.locationType || state.eventType) &&
        mode === TICKET_MODES.SEATED
      ) {
        return {
          ticketMode: TICKET_MODES.GENERAL,
        };
      }

      return {
        ticketMode: mode,
      };
    }),

  // --------------------------------------------------
  // Basic Information
  // --------------------------------------------------

  updateBasicInformation: (data) =>
    set((state) => ({
      basicInformation: {
        ...state.basicInformation,
        ...data,
      },
    })),

  // --------------------------------------------------
  // Date & Time
  // --------------------------------------------------

  updateDateTime: (data) =>
    set((state) => ({
      dateTime: {
        ...state.dateTime,
        ...data,
      },
    })),

  // --------------------------------------------------
  // Event Details
  // --------------------------------------------------

  updateEventDetails: (data) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        ...data,
      },
    })),

  updateContactInformation: (data) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        contactInformation: {
          ...state.eventDetails.contactInformation,
          ...data,
        },
      },
    })),

  updateCancellationPolicy: (policyUpdates) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        cancellationPolicy: {
          ...state.eventDetails.cancellationPolicy,
          ...policyUpdates,
        },
      },
    })),

  updateVisibility: (visibility) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        visibility,
      },
    })),

  // Ticket Types


  addTicketType: (newTicket) =>
    set((state) => ({
      ticketTypes: [
        ...state.ticketTypes,
        {
          id: `ticket-${Date.now()}`,
          status: "ACTIVE",
          ...newTicket,
        },
      ],
    })),

  removeTicketType: (id) =>
    set((state) => ({
      ticketTypes: state.ticketTypes.filter(
        (ticket) => ticket.id !== id
      ),
    })),

  updateTicketType: (id, updatedFields) =>
    set((state) => ({
      ticketTypes: state.ticketTypes.map((ticket) =>
        ticket.id === id
          ? {
            ...ticket,
            ...updatedFields,
          }
          : ticket
      ),
    })),

  updateTicketSalesSettings: (data) =>
    set((state) => ({
      ticketSalesSettings: {
        ...state.ticketSalesSettings,
        ...data,
      },
    })),


  // Seating Configuration


  addCategory: ({
    name,
    price,
    tier = "Standard",
    color = "#00685f",
    rowsCount = 1,
  }) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const cols = cfg.columns?.length
        ? cfg.columns
        : generateDefaultColumns(cfg.seatsPerRow || 8, cfg.columnLabelType || "ALPHA");

      const validRowsCount = Math.max(
        1,
        Math.min(20, Number(rowsCount) || 1)
      );

      const formattedPrice =
        typeof price === "string" && price.startsWith("₹")
          ? price
          : `₹${price || 0}`;

      const newCategory = {
        id: `cat-${Date.now()}`,
        name: name.trim() || "New Tier",
        tier: tier || "Standard",
        price: formattedPrice,
        color: color || "#00685f",
        rows: Array.from(
          { length: validRowsCount },
          (_, rowIndex) => ({
            id: `row-temp-${rowIndex}-${Date.now()}`,
            name: "Row",
            rowLetter: "",
            seats: cols.map((col, seatIndex) => ({
              id: "",
              number: seatIndex + 1,
              columnId: col.id,
              gridCol: col.order_index ?? seatIndex,
              columnLabel: col.label,
              status: "available",
              categoryId: "",
            })),
          })
        ),
      };

      const updatedCategories = rebuildLayout(
        [
          ...cfg.categories,
          newCategory,
        ],
        cols,
        cfg.columnLabelType
      );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

      const prunedSelected = pruneSelectedSeatIds(
        cfg.categories,
        updatedCategories,
        cfg.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...cfg,
          columns: cols,
          seatsPerRow: cols.length,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds: prunedSelected,
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  removeCategory: (categoryId) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      if (
        cfg.categories.length <= 1
      ) {
        return state;
      }

      const remainingCategories =
        cfg.categories.filter(
          (category) => category.id !== categoryId
        );

      const cols = cfg.columns?.length
        ? cfg.columns
        : generateDefaultColumns(cfg.seatsPerRow || 8, cfg.columnLabelType || "ALPHA");

      const updatedCategories = rebuildLayout(
        remainingCategories,
        cols,
        cfg.columnLabelType
      );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

      const prunedSelected = pruneSelectedSeatIds(
        cfg.categories,
        updatedCategories,
        cfg.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...cfg,
          columns: cols,
          seatsPerRow: cols.length,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds: prunedSelected,
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  updateCategory: (categoryId, updatedFields) =>
    set((state) => {
      const updatedCategories =
        state.seatingConfiguration.categories.map(
          (category) => {
            if (category.id === categoryId) {
              return {
                ...category,
                ...updatedFields,
              };
            }

            return category;
          }
        );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
        },
      };
    }),

  setCategoryRowCount: (categoryId, newRowCount) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const validCount = Math.max(
        1,
        Math.min(20, Number(newRowCount) || 1)
      );

      const cols = cfg.columns?.length
        ? cfg.columns
        : generateDefaultColumns(cfg.seatsPerRow || 8, cfg.columnLabelType || "ALPHA");

      const updated = cfg.categories.map((category) => {
        if (category.id !== categoryId) {
          return category;
        }

        const currentRows = category.rows || [];
        if (currentRows.length === validCount) {
          return category;
        }

        let nextRows;
        if (validCount > currentRows.length) {
          const needed = validCount - currentRows.length;
          const newRows = Array.from(
            { length: needed },
            (_, index) => ({
              id: `row-new-${Date.now()}-${index}`,
              name: "",
              rowLetter: "",
              seats: cols.map((col, seatIndex) => ({
                id: "",
                number: seatIndex + 1,
                columnId: col.id,
                gridCol: col.order_index ?? seatIndex,
                columnLabel: col.label,
                status: "available",
                categoryId: category.id,
              })),
            })
          );
          nextRows = [...currentRows, ...newRows];
        } else {
          nextRows = currentRows.slice(0, validCount);
        }

        return {
          ...category,
          rows: nextRows,
        };
      });

      const resequenced = rebuildLayout(
        updated,
        cols,
        cfg.columnLabelType
      );

      const allRows = resequenced.flatMap(
        (category) => category.rows
      );

      const prunedSelected = pruneSelectedSeatIds(
        cfg.categories,
        resequenced,
        cfg.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...cfg,
          columns: cols,
          seatsPerRow: cols.length,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: prunedSelected,
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  addRowToCategory: (categoryId) => {
    const state = get();
    const category = state.seatingConfiguration.categories.find(
      (item) => item.id === categoryId
    );
    if (category) {
      state.setCategoryRowCount(
        categoryId,
        (category.rows?.length || 0) + 1
      );
    }
  },

  removeRowFromCategory: (categoryId, rowId) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const cols = cfg.columns?.length
        ? cfg.columns
        : generateDefaultColumns(cfg.seatsPerRow || 8, cfg.columnLabelType || "ALPHA");

      const updated = cfg.categories.map((category) => {
        if (category.id !== categoryId) {
          return category;
        }

        if (category.rows.length <= 1) {
          return category;
        }

        const nextRows = rowId
          ? category.rows.filter((row) => row.id !== rowId)
          : category.rows.slice(0, -1);

        return {
          ...category,
          rows: nextRows,
        };
      });

      const resequenced = rebuildLayout(
        updated,
        cols,
        cfg.columnLabelType
      );

      const allRows = resequenced.flatMap(
        (category) => category.rows
      );

      const prunedSelected = pruneSelectedSeatIds(
        cfg.categories,
        resequenced,
        cfg.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...cfg,
          columns: cols,
          seatsPerRow: cols.length,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: prunedSelected,
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  generateLayout: (newSeatsPerRow) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const seatsPerRow = Math.max(
        1,
        Math.min(
          30,
          Number(newSeatsPerRow) ||
          cfg.seatsPerRow ||
          8
        )
      );

      const columns = generateDefaultColumns(
        seatsPerRow,
        cfg.columnLabelType || "ALPHA"
      );

      const updatedCategories = cfg.categories.map((category) => {
        const rows = (category.rows || []).map((row) => ({
          ...row,
          seats: columns.map((col, seatIndex) => ({
            id: "",
            number: seatIndex + 1,
            columnId: col.id,
            gridCol: col.order_index,
            columnLabel: col.label,
            status: "available",
            categoryId: category.id,
          })),
        }));

        return {
          ...category,
          rows,
        };
      });

      const resequenced = rebuildLayout(
        updatedCategories,
        columns,
        cfg.columnLabelType
      );

      const allRows = resequenced.flatMap(
        (category) => category.rows
      );

      return {
        seatingConfiguration: {
          ...cfg,
          seatsPerRow,
          columns,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: [],
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  addColumn: (label) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const currentCols = cfg.columns && cfg.columns.length > 0
        ? [...cfg.columns]
        : generateDefaultColumns(cfg.seatsPerRow || 8, cfg.columnLabelType || "ALPHA");

      const newIndex = currentCols.length;
      const autoLabel = cfg.columnLabelType === "NUMERIC"
        ? String(newIndex + 1)
        : getAlphabeticLabel(newIndex);
      const newLabel = label && String(label).trim() ? String(label).trim() : autoLabel;

      const newCol = {
        id: `col-${Date.now()}-${newIndex}`,
        label: newLabel,
        order_index: newIndex,
      };

      const updatedCols = [...currentCols, newCol];
      const resequenced = rebuildLayout(cfg.categories, updatedCols, cfg.columnLabelType);
      const allRows = resequenced.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...cfg,
          columns: updatedCols,
          seatsPerRow: updatedCols.length,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  removeColumn: (columnId) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      if (!cfg.columns || cfg.columns.length <= 1) return state;

      const remainingCols = cfg.columns
        .filter((c) => c.id !== columnId)
        .map((c, idx) => ({
          ...c,
          order_index: idx,
        }));

      const resequenced = rebuildLayout(cfg.categories, remainingCols, cfg.columnLabelType);
      const allRows = resequenced.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...cfg,
          columns: remainingCols,
          seatsPerRow: remainingCols.length,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: [],
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  renameColumn: (columnId, newLabel) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const cleanLabel = String(newLabel || "").trim();
      if (!cleanLabel) return state;

      const updatedCols = (cfg.columns || []).map((c) =>
        c.id === columnId ? { ...c, label: cleanLabel } : c
      );

      const resequenced = rebuildLayout(cfg.categories, updatedCols, cfg.columnLabelType);
      const allRows = resequenced.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...cfg,
          columns: updatedCols,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
        },
      };
    }),

  reorderColumns: (fromIndex, toIndex) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      if (
        !cfg.columns ||
        fromIndex < 0 ||
        toIndex < 0 ||
        fromIndex >= cfg.columns.length ||
        toIndex >= cfg.columns.length ||
        fromIndex === toIndex
      ) {
        return state;
      }

      const cols = [...cfg.columns];
      const [moved] = cols.splice(fromIndex, 1);
      cols.splice(toIndex, 0, moved);

      const reindexed = cols.map((c, idx) => ({
        ...c,
        order_index: idx,
      }));

      const resequenced = rebuildLayout(cfg.categories, reindexed, cfg.columnLabelType);
      const allRows = resequenced.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...cfg,
          columns: reindexed,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
        },
      };
    }),

  setColumnLabelType: (type) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const valid = type === "NUMERIC" ? "NUMERIC" : "ALPHA";

      const updatedCols = (cfg.columns || []).map((col, idx) => ({
        ...col,
        label: valid === "NUMERIC" ? String(idx + 1) : getAlphabeticLabel(idx),
      }));

      const resequenced = rebuildLayout(cfg.categories, updatedCols, valid);
      const allRows = resequenced.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...cfg,
          columnLabelType: valid,
          columns: updatedCols,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
        },
      };
    }),

  toggleSeatGap: (rowId, colOrderIndex) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const updatedCategories = cfg.categories.map((cat) => ({
        ...cat,
        rows: (cat.rows || []).map((r) => {
          if (r.id !== rowId) return r;
          return {
            ...r,
            seats: (r.seats || []).map((s, idx) => {
              const currentGridCol = s.gridCol !== undefined ? s.gridCol : idx;
              if (currentGridCol === colOrderIndex) {
                const nextStatus = s.status === "gap" ? "available" : "gap";
                return {
                  ...s,
                  status: nextStatus,
                };
              }
              return s;
            }),
          };
        }),
      }));

      const allRows = updatedCategories.flatMap((c) => c.rows);
      return {
        seatingConfiguration: {
          ...cfg,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds: cfg.selectedSeatIds.filter((id) => {
            const seat = allRows.flatMap((r) => r.seats).find((s) => s.id === id);
            return seat && seat.status !== "gap";
          }),
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  setSeatStatusAt: (rowId, colOrderIndex, newStatus) =>
    set((state) => {
      const cfg = state.seatingConfiguration;
      const updatedCategories = cfg.categories.map((cat) => ({
        ...cat,
        rows: (cat.rows || []).map((r) => {
          if (r.id !== rowId) return r;
          return {
            ...r,
            seats: (r.seats || []).map((s, idx) => {
              const currentGridCol = s.gridCol !== undefined ? s.gridCol : idx;
              if (currentGridCol === colOrderIndex) {
                return {
                  ...s,
                  status: newStatus,
                };
              }
              return s;
            }),
          };
        }),
      }));

      const allRows = updatedCategories.flatMap((c) => c.rows);
      return {
        seatingConfiguration: {
          ...cfg,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),


  // Temporary Seat Selection


  toggleSeatSelection: (seatId) =>
    set((state) => {
      const currentSelected =
        state.seatingConfiguration.selectedSeatIds || [];

      const isSelected =
        currentSelected.includes(seatId);

      const newSelected = isSelected
        ? currentSelected.filter(
          (id) => id !== seatId
        )
        : [...currentSelected, seatId];

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          selectedSeatIds: newSelected,
        },
      };
    }),


  // Seat Status

  setSelectedSeatsStatus: (newStatus) =>
    set((state) => {
      const selectedIds = new Set(
        state.seatingConfiguration.selectedSeatIds
      );

      const updatedCategories =
        state.seatingConfiguration.categories.map(
          (category) => ({
            ...category,
            rows: category.rows.map((row) => ({
              ...row,
              seats: row.seats.map((seat) => {
                if (selectedIds.has(seat.id)) {
                  return {
                    ...seat,
                    status: newStatus,
                  };
                }

                return seat;
              }),
            })),
          })
        );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds: [],
          totalCapacity: allRows.reduce(
            (sum, row) => sum + (row.seats || []).filter((s) => s.status !== "gap").length,
            0
          ),
        },
      };
    }),

  // Assign Category To Selected Seats


  assignCategoryToSelectedSeats: (categoryId) =>
    set((state) => {
      const selectedIds = new Set(
        state.seatingConfiguration.selectedSeatIds
      );

      const updatedCategories =
        state.seatingConfiguration.categories.map(
          (category) => ({
            ...category,
            rows: category.rows.map((row) => ({
              ...row,
              seats: row.seats.map((seat) => {
                if (selectedIds.has(seat.id)) {
                  return {
                    ...seat,
                    categoryId,
                  };
                }

                return seat;
              }),
            })),
          })
        );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
        },
      };
    }),


  // Generic Step Data


  updateStepData: (stepKey, data) =>
    set((state) => ({
      [stepKey]: {
        ...state[stepKey],
        ...data,
      },
    })),


  // Payout Account

  // updatePayoutAccount: (accountData) =>
  //   set((state) => ({
  //     payoutAccount: {
  //       ...state.payoutAccount,
  //       ...accountData,
  //     },
  //   })),


  syncSavedSeatLayout: (payload) => {
    if (!payload) return;
    const colors = [
      "#00685f",
      "#4648d4",
      "#6d7a77",
      "#d97706",
      "#9333ea",
      "#e11d48",
    ];

    const data = payload?.data || payload;
    const layout = data?.layout || data?.SeatLayout || data?.Layout || {};
    const layoutName =
      layout.LayoutName || layout.layout_name || "Main Seating Layout";

    const rawSections =
      data.sections || data.Sections || layout.sections || layout.Sections || [];
    const rawRows = data.rows || data.Rows || [];
    const rawSeats = data.seats || data.Seats || [];
    const rawCols =
      data.columns || data.Columns || layout.columns || layout.Columns || [];

    let columns = [];
    if (Array.isArray(rawCols) && rawCols.length > 0) {
      columns = rawCols.map((c, idx) => ({
        id: c.ID || c.id || `col-${idx}`,
        label: c.Label || c.label || getAlphabeticLabel(idx),
        order_index:
          c.OrderIndex !== undefined
            ? c.OrderIndex
            : c.order_index !== undefined
            ? c.order_index
            : idx,
      })).sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0));
    }

    const isFlat =
      Array.isArray(rawRows) &&
      rawRows.length > 0 &&
      rawSections.every((s) => !Array.isArray(s.rows) && !Array.isArray(s.Rows));

    let categories = [];

    if (isFlat) {
      const seatsByRowId = new Map();
      rawSeats.forEach((seat) => {
        const rowId = seat.RowID || seat.row_id || seat.rowId;
        if (!seatsByRowId.has(rowId)) {
          seatsByRowId.set(rowId, []);
        }
        seatsByRowId.get(rowId).push({
          id: seat.ID || seat.id,
          number: seat.SeatNumber || seat.seat_number || seat.seatNumber,
          columnId: seat.ColumnID || seat.column_id || seat.columnId,
          gridCol:
            seat.GridCol !== undefined
              ? seat.GridCol
              : seat.grid_col !== undefined
              ? seat.grid_col
              : undefined,
          columnLabel: seat.ColumnLabel || seat.column_label || "",
          status:
            String(seat.Status || seat.status || "").toUpperCase() ===
            "DISABLED"
              ? "disabled"
              : "available",
          rowId,
        });
      });

      if (columns.length === 0) {
        let maxCols = 8;
        rawSeats.forEach((s) => {
          if (s.gridCol !== undefined && s.gridCol + 1 > maxCols) {
            maxCols = s.gridCol + 1;
          }
        });
        columns = generateDefaultColumns(maxCols, "ALPHA");
      }

      const rowsBySectionId = new Map();
      rawRows.forEach((row) => {
        const secId = row.SectionID || row.section_id || row.sectionId;
        if (!rowsBySectionId.has(secId)) {
          rowsBySectionId.set(secId, []);
        }
        const rowId = row.ID || row.id;
        const rowSeatsRaw = seatsByRowId.get(rowId) || [];
        const seatByGridCol = new Map();
        rowSeatsRaw.forEach((s) => {
          if (s.columnId) seatByGridCol.set(`col-${s.columnId}`, s);
          if (s.gridCol !== undefined) seatByGridCol.set(`grid-${s.gridCol}`, s);
        });

        const alignedSeats = columns.map((col, colIdx) => {
          const found =
            seatByGridCol.get(`col-${col.id}`) ||
            seatByGridCol.get(`grid-${col.order_index ?? colIdx}`);
          if (found) {
            return {
              ...found,
              columnId: col.id,
              gridCol: col.order_index ?? colIdx,
              columnLabel: col.label,
              categoryId: secId,
            };
          }
          return {
            id: `gap-${rowId}-${colIdx}`,
            number: colIdx + 1,
            columnId: col.id,
            gridCol: col.order_index ?? colIdx,
            columnLabel: col.label,
            status: "gap",
            categoryId: secId,
            rowId,
          };
        });

        rowsBySectionId.get(secId).push({
          id: rowId,
          name: `Row ${row.RowName || row.row_name || row.rowName}`,
          rowLetter: row.RowName || row.row_name || row.rowName,
          seats: alignedSeats,
        });
      });

      categories = rawSections.map((sec, idx) => {
        const rawPrice = sec.Price ?? sec.price ?? 0;
        const formattedPrice =
          typeof rawPrice === "string" && rawPrice.startsWith("₹")
            ? rawPrice
            : `₹${rawPrice}`;
        const secId = sec.ID || sec.id;
        return {
          id: secId,
          name: sec.Name || sec.name || "",
          tier: sec.Name || sec.name || "Standard",
          price: formattedPrice,
          color: colors[idx % colors.length],
          rows: rowsBySectionId.get(secId) || [],
        };
      });
    } else {
      if (columns.length === 0) {
        let maxCols = 8;
        rawSections.forEach((sec) => {
          (sec.Rows || sec.rows || []).forEach((r) => {
            const seats = r.Seats || r.seats || [];
            if (seats.length > maxCols) maxCols = seats.length;
          });
        });
        columns = generateDefaultColumns(maxCols, "ALPHA");
      }

      categories = rawSections.map((sec, idx) => {
        const rawPrice = sec.Price ?? sec.price ?? 0;
        const formattedPrice =
          typeof rawPrice === "string" && rawPrice.startsWith("₹")
            ? rawPrice
            : `₹${rawPrice}`;
        const rows = sec.Rows || sec.rows || [];
        const secId = sec.ID || sec.id;
        return {
          id: secId,
          name: sec.Name || sec.name || "",
          tier: sec.Name || sec.name || "Standard",
          price: formattedPrice,
          color: colors[idx % colors.length],
          rows: rows.map((row) => {
            const rawRowSeats = row.Seats || row.seats || [];
            const seatByGridCol = new Map();
            rawRowSeats.forEach((s, sIdx) => {
              const cId = s.ColumnID || s.column_id || s.columnId;
              const gCol =
                s.GridCol !== undefined
                  ? s.GridCol
                  : s.grid_col !== undefined
                  ? s.grid_col
                  : sIdx;
              if (cId) seatByGridCol.set(`col-${cId}`, s);
              seatByGridCol.set(`grid-${gCol}`, s);
            });

            const alignedSeats = columns.map((col, colIdx) => {
              const found =
                seatByGridCol.get(`col-${col.id}`) ||
                seatByGridCol.get(`grid-${col.order_index ?? colIdx}`);
              if (found) {
                return {
                  id: found.ID || found.id,
                  number:
                    found.SeatNumber ||
                    found.seat_number ||
                    found.seatNumber ||
                    found.number ||
                    colIdx + 1,
                  columnId: col.id,
                  gridCol: col.order_index ?? colIdx,
                  columnLabel: col.label,
                  status:
                    String(found.Status || found.status || "").toUpperCase() ===
                    "DISABLED"
                      ? "disabled"
                      : "available",
                  categoryId: secId,
                };
              }
              return {
                id: `gap-${row.ID || row.id}-${colIdx}`,
                number: colIdx + 1,
                columnId: col.id,
                gridCol: col.order_index ?? colIdx,
                columnLabel: col.label,
                status: "gap",
                categoryId: secId,
              };
            });

            return {
              id: row.ID || row.id,
              name: `Row ${row.RowName || row.row_name || row.rowName}`,
              rowLetter: row.RowName || row.row_name || row.rowName,
              seats: alignedSeats,
            };
          }),
        };
      });
    }

    const allRows = categories.flatMap((cat) => cat.rows || []);
    const totalCapacity = allRows.reduce(
      (sum, r) => sum + (r.seats || []).filter((s) => s.status !== "gap").length,
      0
    );
    const seatsPerRow = columns.length;

    const isNumericLabels =
      columns.length > 0 &&
      columns.every((c) => /^\d+$/.test(String(c.label || "").trim()));
    const inferredLabelType = isNumericLabels ? "NUMERIC" : "ALPHA";

    set((state) => ({
      seatingConfiguration: {
        ...state.seatingConfiguration,
        layoutName,
        columns,
        columnLabelType: inferredLabelType,
        categories,
        sections: categories,
        rows: allRows,
        seatsPerRow,
        selectedSeatIds: [],
        totalCapacity,
      },
    }));
  },

  hydrateForEdit: (data) => {
    if (!data) return;

    const event = data.Event || {};
    const schedule = data.Schedule || {};
    const venue = data.Venue || {};
    const setting = data.Setting || {};
    const cancellation = data.Cancellation || {};
    const contact = data.Contact || {};
    const ticketTypes = data.TicketTypes || [];
    const seatLayout = data.SeatLayout || null;

    const isOnline = event.EventType === LOCATION_TYPES.ONLINE;
    const ticketMode =
      setting.SeatLayoutType || (isOnline ? TICKET_MODES.GENERAL : TICKET_MODES.GENERAL);

    // Format dates to YYYY-MM-DD
    const formatDate = (val) => (val ? String(val).slice(0, 10) : "");
    // Format times to HH:mm
    const formatTime = (val) => (val ? String(val).slice(0, 5) : "");

    // Hydrate seating categories if seat layout exists
    let seatingConfig = { ...initialSeatingConfig };
    if (
      seatLayout &&
      Array.isArray(seatLayout.Sections) &&
      seatLayout.Sections.length > 0
    ) {
      const colors = [
        "#00685f",
        "#4648d4",
        "#6d7a77",
        "#d97706",
        "#9333ea",
        "#e11d48",
      ];

      const rawCols =
        seatLayout.Columns || seatLayout.columns || [];
      let columns = [];
      if (Array.isArray(rawCols) && rawCols.length > 0) {
        columns = rawCols.map((c, idx) => ({
          id: c.ID || c.id || `col-${idx}`,
          label: c.Label || c.label || getAlphabeticLabel(idx),
          order_index:
            c.OrderIndex !== undefined
              ? c.OrderIndex
              : c.order_index !== undefined
              ? c.order_index
              : idx,
        })).sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0));
      }

      const rawSections = seatLayout.Sections || seatLayout.sections || [];

      if (columns.length === 0) {
        let maxCols = 8;
        rawSections.forEach((sec) => {
          (sec.Rows || sec.rows || []).forEach((r) => {
            const seats = r.Seats || r.seats || [];
            if (seats.length > maxCols) maxCols = seats.length;
          });
        });
        columns = generateDefaultColumns(maxCols, "ALPHA");
      }

      const categories = rawSections.map((sec, idx) => {
        const secId = sec.ID || sec.id;
        const secName = sec.Name || sec.name || "";
        const rawPrice = sec.Price ?? sec.price ?? 0;
        const formattedPrice =
          typeof rawPrice === "string" && rawPrice.startsWith("₹")
            ? rawPrice
            : `₹${rawPrice}`;

        return {
          id: secId,
          name: secName,
          tier: secName || "Standard",
          price: formattedPrice,
          color: colors[idx % colors.length],
          rows: (sec.Rows || sec.rows || []).map((row) => {
            const rawSeats = row.Seats || row.seats || [];
            const rowId = row.ID || row.id;
            const rowLetter = row.RowName || row.row_name || row.rowLetter || "";
            const seatByGridCol = new Map();
            rawSeats.forEach((s, sIdx) => {
              const cId = s.ColumnID || s.column_id || s.columnId;
              const gCol =
                s.GridCol !== undefined
                  ? s.GridCol
                  : s.grid_col !== undefined
                  ? s.grid_col
                  : sIdx;
              if (cId) seatByGridCol.set(`col-${cId}`, s);
              seatByGridCol.set(`grid-${gCol}`, s);
            });

            const alignedSeats = columns.map((col, colIdx) => {
              const found =
                seatByGridCol.get(`col-${col.id}`) ||
                seatByGridCol.get(`grid-${col.order_index ?? colIdx}`);
              if (found) {
                return {
                  id: found.ID || found.id,
                  number:
                    found.SeatNumber ||
                    found.seat_number ||
                    found.seatNumber ||
                    found.number ||
                    colIdx + 1,
                  columnId: col.id,
                  gridCol: col.order_index ?? colIdx,
                  columnLabel: col.label,
                  status:
                    String(found.Status || found.status || "").toUpperCase() ===
                    "DISABLED"
                      ? "disabled"
                      : "available",
                  categoryId: secId,
                };
              }
              return {
                id: `gap-${rowId}-${colIdx}`,
                number: colIdx + 1,
                columnId: col.id,
                gridCol: col.order_index ?? colIdx,
                columnLabel: col.label,
                status: "gap",
                categoryId: secId,
              };
            });

            return {
              id: rowId,
              name: `Row ${rowLetter}`,
              rowLetter: rowLetter,
              seats: alignedSeats,
            };
          }),
        };
      });

      const allRows = categories.flatMap((cat) => cat.rows || []);
      const totalCapacity = allRows.reduce(
        (sum, r) => sum + (r.seats || []).filter((s) => s.status !== "gap").length,
        0
      );

      const isNumericLabels =
        columns.length > 0 &&
        columns.every((c) => /^\d+$/.test(String(c.label || "").trim()));
      const inferredLabelType = isNumericLabels ? "NUMERIC" : "ALPHA";

      seatingConfig = {
        layoutName:
          seatLayout.LayoutName || seatLayout.layout_name || "Main Seating Layout",
        columns,
        columnLabelType: inferredLabelType,
        categories,
        sections: categories,
        rows: allRows,
        seatsPerRow: columns.length,
        selectedSeatIds: [],
        totalCapacity,
      };
    }

    set({
      isEditMode: true,
      editingEventId: event.ID || null,

      locationType: event.EventType || LOCATION_TYPES.PHYSICAL,
      eventType: event.EventType || LOCATION_TYPES.PHYSICAL,
      onlineUrl: event.OnlineURL || "",

      venue: isOnline
        ? { ...initialVenue }
        : {
            google_place_id: venue.GooglePlaceID || "",
            name: venue.Name || "",
            address: venue.Address || "",
            city: venue.City || "",
            state: venue.State || "",
            country: venue.Country || "",
            postal_code: venue.PostalCode || "",
            latitude: venue.Latitude ?? null,
            longitude: venue.Longitude ?? null,
          },

      basicInformation: {
        title: event.Title || "",
        category: event.CategoryName || "",
        categoryId: event.CategoryID || "",
        description: event.Description || "",
        banner: event.BannerURL || "",
        bannerFile: null,
        bannerPreviewUrl: event.BannerImageURL || event.BannerURL || "",
        originalBannerPreviewUrl: event.BannerImageURL || event.BannerURL || "",
        language: event.Language || "",
        ageRestriction:
          event.AgeRestriction !== undefined && event.AgeRestriction !== null
            ? String(event.AgeRestriction)
            : "",
      },

      dateTime: {
        eventDate: formatDate(schedule.EventDate),
        startTime: formatTime(schedule.StartTime),
        endTime: formatTime(schedule.EndTime),
        isAllDay: Boolean(schedule.IsAllDay),
      },

      eventDetails: {
        highlights: event.Highlights || "",
        rules: event.Rules || "",
        contactInformation: {
          name: contact.Name || "",
          phone: contact.Phone || "",
          email: contact.Email || "",
        },
        cancellationPolicy: {
          allowCancellation: Boolean(cancellation.CancellationAllowed),
          cancellationDeadline: mapHoursToCancellationDeadline(
            cancellation.CancellationDeadlineHours
          ),
          refundPolicy: cancellation.RefundPolicy || "PARTIAL",
          refundPercentage: cancellation.RefundPercentage ?? 80,
          organizerPolicyAccepted: true,
        },
        attendeeInformation: event.AttendeeInformation || "",
        visibility: event.Visibility || "PUBLIC",
      },

      ticketMode,

      ticketTypes: ticketTypes.map((ticket) => ({
        id: ticket.ID,
        name: ticket.Name || "",
        price: ticket.Price || 0,
        capacity: ticket.TotalQuantity || 0,
        description: ticket.Description || "",
      })),

      ticketSalesSettings: {
        salesStartDate: formatDate(setting.SalesStartDate),
        salesEndDate: formatDate(setting.SalesEndDate),
        maxTicketsPerBooking: setting.BookingLimitPerUser
          ? String(setting.BookingLimitPerUser)
          : "",
      },

      seatingConfiguration: seatingConfig,
    });
  },

  resetForm: () => {
    const currentPreview =
      get().basicInformation?.bannerPreviewUrl;

    if (currentPreview) {
      URL.revokeObjectURL(currentPreview);
    }

    set({
      isEditMode: false,
      editingEventId: null,
      locationType: LOCATION_TYPES.PHYSICAL,
      eventType: LOCATION_TYPES.PHYSICAL,
      onlineUrl: "",
      venue: { ...initialVenue },
      basicInformation: { ...initialBasicInfo },
      dateTime: { ...initialDateTime },
      eventDetails: { ...initialEventDetails },
      ticketMode: TICKET_MODES.GENERAL,
      ticketTypes: [],
      ticketSalesSettings: {
        ...initialTicketSalesSettings,
      },
      seatingConfiguration: {
        ...initialSeatingConfig,
      },
      // payoutAccount: {
      //   ...initialPayoutAccount,
      // },
    });
  },
}));

export default useEventCreationStore;