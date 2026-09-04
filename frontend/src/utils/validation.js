export function validateFullName(name) {
    const value = name.trim();

    if (!value) {
        return "Full name is required";
    }

    if (value.length < 2) {
        return "Full name must be at least 2 characters";
    }

    if (value.length > 50) {
        return "Full name must not exceed 50 characters";
    }

    // Only letters and spaces
    if (!/^[A-Za-z]+(?: [A-Za-z]+)*$/.test(value)) {
        return "Full name can contain only letters and spaces";
    }

    return "";
}

export function validateEmail(email) {
    const value = email.trim();

    if (!value) {
        return "Email is required";
    }

    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
        return "Enter a valid email address";
    }

    return "";
}

export function validatePhone(phone) {
    const value = phone.trim();

    if (!value) {
        return "Phone number is required";
    }

    if (!/^\d{10}$/.test(value)) {
        return "Phone number must contain exactly 10 digits";
    }

    return "";
}

export function validatePassword(password) {
    if (!password) {
        return "Password is required";
    }

    if (password.length < 8) {
        return "Password must be at least 8 characters";
    }

    if (!/[A-Z]/.test(password)) {
        return "Password must contain at least one uppercase letter";
    }

    if (!/[a-z]/.test(password)) {
        return "Password must contain at least one lowercase letter";
    }

    if (!/[0-9]/.test(password)) {
        return "Password must contain at least one number";
    }

    if (!/[^A-Za-z0-9]/.test(password)) {
        return "Password must contain at least one special character";
    }

    return "";
}

export function validatePAN(pan) {
    const value = pan?.trim().toUpperCase() || "";

    if (!value) {
        return "PAN number is required";
    }

    if (!/^[A-Z]{5}[0-9]{4}[A-Z]{1}$/.test(value)) {
        return "Enter a valid 10-character PAN number (e.g. ABCDE1234F)";
    }

    return "";
}

export function validateGST(gst) {
    const value = gst?.trim().toUpperCase() || "";

    if (!value) {
        return ""; // Optional field
    }

    if (!/^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$/.test(value)) {
        return "Enter a valid 15-character GST number (e.g. 33ABCDE1234F1Z5)";
    }

    return "";
}

export function validateIFSC(ifsc) {
    const value = ifsc?.trim().toUpperCase() || "";

    if (!value) {
        return "IFSC code is required";
    }

    if (!/^[A-Z]{4}0[A-Z0-9]{6}$/.test(value)) {
        return "Enter a valid 11-character IFSC code (e.g. SBIN0001234)";
    }

    return "";
}

export function validateURL(url) {
    const value = url?.trim() || "";

    if (!value) {
        return ""; // Optional field
    }

    try {
        const parsed = new URL(value.startsWith("http") ? value : `https://${value}`);
        if (!parsed.hostname || !parsed.hostname.includes(".")) {
            return "Enter a valid website URL";
        }
    } catch {
        return "Enter a valid website URL";
    }

    return "";
}

export function maskPAN(pan) {
    if (!pan || pan.length < 10) return pan || "—";
    return `${pan.slice(0, 3)}******${pan.slice(-1)}`;
}

export function maskAccountNumber(acc) {
    if (!acc) return "—";
    const last4 = acc.slice(-4);
    return `•••• •••• ${last4}`;
}