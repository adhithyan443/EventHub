import apiClient from "./client";

/**
 * Retrieves all tickets belonging to the authenticated customer.
 * @returns {Promise<{ data: Array }>}
 */
export const getMyTickets = async () => {
  const response = await apiClient.get("/tickets");
  return response.data;
};

/**
 * Retrieves a single ticket belonging to the authenticated customer.
 * @param {string} ticketId - UUID of the ticket
 * @returns {Promise<{ data: object }>}
 */
export const getMyTicketById = async (ticketId) => {
  if (!ticketId) {
    throw new Error("Ticket ID is required.");
  }

  const response = await apiClient.get(`/tickets/${ticketId}`);
  return response.data;
};

/**
 * Downloads a customer ticket PDF pass as a binary blob and triggers browser download.
 * @param {string} ticketId - UUID of the ticket
 * @returns {Promise<{ filename: string }>}
 */
export const downloadTicketPdf = async (ticketId) => {
  if (!ticketId) {
    throw new Error("Ticket ID is required to download ticket PDF.");
  }

  try {
    const response = await apiClient.get(`/tickets/${ticketId}/download`, {
      responseType: "blob",
    });

    let filename = `ticket-${ticketId}.pdf`;
    const disposition = response.headers?.["content-disposition"];
    if (disposition) {
      const match = disposition.match(/filename=["']?([^"';]+)["']?/i);
      if (match && match[1]) {
        filename = match[1];
      }
    }

    const blob = new Blob([response.data], { type: "application/pdf" });
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.setAttribute("download", filename);
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(downloadUrl);

    return { filename };
  } catch (err) {
    if (err.response?.data instanceof Blob) {
      try {
        const text = await err.response.data.text();
        const parsed = JSON.parse(text);
        if (parsed?.error || parsed?.message) {
          err.message = parsed.error || parsed.message;
        }
      } catch {
        // Fallback to generic message
      }
    }
    throw err;
  }
};
