import apiClient from "./client";

/**
 * Creates a payment order for a persistent checkout.
 * @param {object} params
 * @param {string} params.checkoutId - UUID of the active checkout
 * @param {string} params.method - Payment method ("UPI", "CARD", "NET_BANKING", "WALLET")
 * @returns {Promise<object>} - Envelope containing { success: true, data: { payment, keyId } }
 */
export const createPayment = async ({ checkoutId, method }) => {
    if (!checkoutId) {
        throw new Error("Checkout ID is required to create a payment.");
    }

    const response = await apiClient.post("/payments", {
        checkoutId,
        method: method || "UPI",
    });

    return response.data;
};

/**
 * Verifies a Razorpay payment cryptographically and confirms the booking.
 * @param {object} params
 * @param {string} params.providerOrderId - Razorpay Order ID (e.g., "order_xxx")
 * @param {string} params.razorpayPaymentId - Razorpay Payment ID (e.g., "pay_xxx")
 * @param {string} params.razorpaySignature - HMAC SHA256 signature from Razorpay checkout modal
 * @returns {Promise<object>} - Envelope containing { success: true, data: { payment, booking } }
 */
export const verifyPayment = async ({
    providerOrderId,
    razorpayPaymentId,
    razorpaySignature,
}) => {
    if (!providerOrderId || !razorpayPaymentId || !razorpaySignature) {
        throw new Error("Missing required parameters for payment verification.");
    }

    const response = await apiClient.post("/payments/verify", {
        providerOrderId,
        razorpayPaymentId,
        razorpaySignature,
    });

    return response.data;
};
