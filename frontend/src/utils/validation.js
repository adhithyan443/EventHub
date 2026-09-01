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