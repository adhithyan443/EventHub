import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { startGoogleLogin, register } from "../../api/authApi";

import AuthLayout from "../../components/layout/AuthLayout";
import Input from "../../components/ui/Input";
import PasswordInput from "../../components/ui/PasswordInput";
import PasswordStrength from "../../components/ui/PasswordStrength";
import Checkbox from "../../components/ui/Checkbox";
import Button from "../../components/ui/Button";
import GoogleIcon from "../../components/ui/GoogleIcon";

import {
    validateFullName,
    validateEmail,
    validatePhone,
    validatePassword,
} from "../../utils/validation";

export default function RegisterPage() {
    const navigate = useNavigate();

    const [form, setForm] = useState({
        fullName: "",
        email: "",
        phone: "",
        password: "",
        confirmPassword: "",
        agree: false,
    });

    const [errors, setErrors] = useState({});
    const [isSubmitting, setIsSubmitting] = useState(false);

    function handleChange(e) {
        const { name, value, type, checked } = e.target;

        setForm((prev) => ({
            ...prev,
            [name]: type === "checkbox" ? checked : value,
        }));

        // Clear field error when user starts correcting it
        setErrors((prev) => ({
            ...prev,
            [name]: "",
            submit: "",
        }));
    }

    function validateForm() {
        const newErrors = {};

        const nameError = validateFullName(form.fullName);
        const emailError = validateEmail(form.email);
        const phoneError = validatePhone(form.phone);
        const passwordError = validatePassword(form.password);

        if (nameError) {
            newErrors.fullName = nameError;
        }

        if (emailError) {
            newErrors.email = emailError;
        }

        if (phoneError) {
            newErrors.phone = phoneError;
        }

        if (passwordError) {
            newErrors.password = passwordError;
        }

        if (!form.confirmPassword) {
            newErrors.confirmPassword = "Please confirm your password";
        } else if (form.confirmPassword !== form.password) {
            newErrors.confirmPassword = "Passwords do not match";
        }

        if (!form.agree) {
            newErrors.agree =
                "You must agree to the Terms and Privacy Policy";
        }

        setErrors(newErrors);

        return Object.keys(newErrors).length === 0;
    }

    async function handleSubmit(e) {
        e.preventDefault();

        const isValid = validateForm();

        if (!isValid) {
            return;
        }

        setIsSubmitting(true);

        try {
            await register({
                fullName: form.fullName.trim(),
                email: form.email.trim().toLowerCase(),
                phone: form.phone.trim(),
                password: form.password,
            });

            navigate("/verify-otp", {
                replace: true,
                state: {
                    email: form.email.trim().toLowerCase(),
                },
            });
        } catch (error) {
            console.error("Registration failed:", error);

            const message =
                error.response?.data?.message ||
                "Registration failed. Please try again.";

            setErrors({
                submit: message,
            });
        } finally {
            setIsSubmitting(false);
        }
    }

    return (
        <AuthLayout
            title={
                <>
                    Discover. Book.
                    <br />
                    Experience.
                </>
            }
            description="Create your account and start discovering amazing events."
        >
            <div className="bg-white rounded-lg shadow p-8 w-full max-w-md flex flex-col gap-6">
                <div>
                    <h2 className="font-display text-2xl font-semibold text-ink">
                        Create your account
                    </h2>

                    <p className="text-ink/60 text-sm mt-1">
                        Join EventHub and start discovering amazing events.
                    </p>
                </div>

                <form
                    onSubmit={handleSubmit}
                    className="flex flex-col gap-4"
                    noValidate
                >
                    {/* Full Name */}
                    <Input
                        label="Full Name"
                        name="fullName"
                        placeholder="Enter your full name"
                        value={form.fullName}
                        onChange={handleChange}
                        error={errors.fullName}
                    />

                    {/* Email */}
                    <Input
                        label="Email Address"
                        name="email"
                        type="email"
                        placeholder="Enter your email"
                        value={form.email}
                        onChange={handleChange}
                        error={errors.email}
                    />

                    {/* Phone */}
                    <Input
                        label="Phone Number"
                        name="phone"
                        type="tel"
                        placeholder="Enter your phone number"
                        leftAddon="+91"
                        value={form.phone}
                        onChange={handleChange}
                        error={errors.phone}
                    />

                    {/* Password */}
                    <div>
                        <PasswordInput
                            label="Password"
                            name="password"
                            placeholder="Create a password"
                            value={form.password}
                            onChange={handleChange}
                            error={errors.password}
                        />

                        <PasswordStrength password={form.password} />
                    </div>

                    {/* Confirm Password */}
                    <PasswordInput
                        label="Confirm Password"
                        name="confirmPassword"
                        placeholder="Re-enter your password"
                        value={form.confirmPassword}
                        onChange={handleChange}
                        error={errors.confirmPassword}
                    />

                    {/* Terms */}
                    <Checkbox
                        name="agree"
                        checked={form.agree}
                        onChange={handleChange}
                        error={errors.agree}
                    >
                        I agree to the{" "}
                        <a
                            href="/terms"
                            className="text-primary font-medium"
                        >
                            Terms of Service
                        </a>{" "}
                        and{" "}
                        <a
                            href="/privacy"
                            className="text-primary font-medium"
                        >
                            Privacy Policy
                        </a>
                        .
                    </Checkbox>

                    {/* Server Error */}
                    {errors.submit && (
                        <p className="text-sm text-red-500">
                            {errors.submit}
                        </p>
                    )}

                    {/* Submit */}
                    <Button
                        type="submit"
                        disabled={isSubmitting}
                    >
                        {isSubmitting
                            ? "Creating Account..."
                            : "Create Account"}
                    </Button>
                </form>

                {/* Divider */}
                <div className="flex items-center gap-4">
                    <div className="flex-1 h-px bg-border" />

                    <span className="text-xs font-medium tracking-wider text-ink/60 uppercase">
                        OR
                    </span>

                    <div className="flex-1 h-px bg-border" />
                </div>

                {/* Google Login */}
                <Button
                    variant="outline"
                    icon={<GoogleIcon />}
                    onClick={startGoogleLogin}
                >
                    Continue with Google
                </Button>

                {/* Login */}
                <p className="text-center text-sm text-ink/60">
                    Already have an account?{" "}
                    <Link
                        to="/login"
                        className="text-primary font-semibold"
                    >
                        Sign in
                    </Link>
                </p>
            </div>
        </AuthLayout>
    );
}