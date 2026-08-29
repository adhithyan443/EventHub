import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import AuthLayout from "../../components/layout/AuthLayout";
// import Input from "../../components/ui/Input";
import PasswordInput from "../../components/ui/PasswordInput";
import PasswordStrength from "../../components/ui/PasswordStrength";
import Button from "../../components/ui/Button";
import { resetPassword } from "../../api/authApi";

export default function ResetPasswordPage() {
    // Gets the reset token from the link received through email.
    const [searchParams] = useSearchParams();

    // Token is kept internally and is NOT displayed to the user.
    const token = searchParams.get("token") || "";

    const [password, setPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");

    const [errors, setErrors] = useState({});
    // const [success, setSuccess] = useState("");
    const [loading, setLoading] = useState(false);

    const navigate = useNavigate();

    function handleSubmit(e) {
        e.preventDefault();

        const newErrors = {};

        if (!token) {
            newErrors.general = "Invalid or missing reset link.";
        }

        if (!password) {
            newErrors.password = "Password is required";
        }

        if (password.length < 8) {
            newErrors.password = "Password must contain at least 8 characters";
        }

        if (!confirmPassword) {
            newErrors.confirmPassword = "Please confirm your password";
        } else if (password !== confirmPassword) {
            newErrors.confirmPassword = "Passwords do not match";
        }

        setErrors(newErrors);

        if (Object.keys(newErrors).length > 0) {
            return;
        }

        handleResetPassword();
    }

    async function handleResetPassword() {
        try {
            setLoading(true);
            setErrors({});
            // setSuccess("");

            const response = await resetPassword(token, password);

            console.log("Password reset successful:", response);

            // setSuccess(
            //     "Password reset successfully. Redirecting to login..."
            // );

            setTimeout(() => {
                navigate("/login");
            }, 1500);

        } catch (error) {
            console.error("Password reset failed:", error);

            const message =
                error.response?.data?.message ||
                "Unable to reset password. Please try again.";

            setErrors({
                general: message,
            });
        } finally {
            setLoading(false);
        }
    }

    return (
        <AuthLayout
            title={
                <>
                    Discover.<br />
                    Book.<br />
                    Experience.
                </>
            }
            description="Create a new password for your EventHub account."
        >
            <div className="bg-white rounded-lg shadow p-8 w-full max-w-md flex flex-col gap-6">

                <div>
                    <h2 className="font-display text-2xl font-semibold text-ink">
                        Reset Password
                    </h2>

                    <p className="text-ink/60 text-sm mt-1">
                        Create a new password for your account.
                    </p>
                </div>

                {/* General error */}
                {errors.general && (
                    <p className="text-sm text-red-500 text-center">
                        {errors.general}
                    </p>
                )}


                <form
                    onSubmit={handleSubmit}
                    className="flex flex-col gap-5"
                >

                    {/* NEW PASSWORD */}
                    <div>
                        <PasswordInput
                            label="New Password"
                            name="password"
                            placeholder="Create a new password"
                            value={password}
                            onChange={(e) => {
                                setPassword(e.target.value);

                                // Clear password error while typing.
                                setErrors((prev) => ({
                                    ...prev,
                                    password: "",
                                    general: "",
                                }));
                            }}
                            error={errors.password}
                        />

                        {/* Password strength indicator */}
                        <PasswordStrength password={password} />
                    </div>

                    {/* CONFIRM PASSWORD */}
                    <PasswordInput
                        label="Confirm Password"
                        name="confirmPassword"
                        placeholder="Re-enter your password"
                        value={confirmPassword}
                        onChange={(e) => {
                            setConfirmPassword(e.target.value);

                            // Clear confirm password error while typing.
                            setErrors((prev) => ({
                                ...prev,
                                confirmPassword: "",
                                general: "",
                            }));
                        }}
                        error={errors.confirmPassword}
                    />

                    {/* SUBMIT */}
                    <Button type="submit" disabled={loading}>
                        {loading ? "Resetting..." : "Reset Password"}
                    </Button>
                </form>

                <div className="text-center">
                    <Link
                        to="/login"
                        className="text-sm text-primary font-semibold"
                    >
                        Back to Login
                    </Link>
                </div>
            </div>
        </AuthLayout>
    );
}