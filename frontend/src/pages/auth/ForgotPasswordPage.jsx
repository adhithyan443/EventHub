import { useState } from "react";
import { Link } from "react-router-dom";
import AuthLayout from "../../components/layout/AuthLayout";
import Input from "../../components/ui/Input";
import Button from "../../components/ui/Button";
import { forgotPassword } from "../../api/authApi";

export default function ForgotPasswordPage() {
    const [email, setEmail] = useState("");
    const [error, setError] = useState("");
    const [message, setMessage] = useState("");
    const [loading, setLoading] = useState(false);



    async function handleSubmit(e) {
        e.preventDefault();

        setError("");
        setMessage("");

        if (!email.trim()) {
            setError("Email is required");
            return;
        }

        try {
            setLoading(true);

            const response = await forgotPassword(email);

            console.log("Forgot password response:", response);

            setMessage(response.message);

        } catch (error) {
            console.error("Forgot password failed:", error);

            setError(
                error.response?.data?.message ||
                "Unable to process your request. Please try again."
            );
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
            description="Reset your EventHub password and get back to discovering amazing events."
        >
            <div className="bg-white rounded-lg shadow p-8 w-full max-w-md flex flex-col gap-6">

                <div>
                    <h2 className="font-display text-2xl font-semibold text-ink">
                        Forgot Password?
                    </h2>

                    <p className="text-ink/60 text-sm mt-1">
                        Enter your email address and we'll send you a password
                        reset instruction.
                    </p>
                </div>

                {error && (
                    <p className="text-sm text-red-500 text-center">
                        {error}
                    </p>
                )}

                {message && (
                    <p className="text-sm text-green-600 text-center">
                        {message}
                    </p>
                )}

                <form
                    onSubmit={handleSubmit}
                    className="flex flex-col gap-6"
                >
                    <Input
                        label="Email Address"
                        name="email"
                        type="email"
                        placeholder="Enter your email"
                        value={email}
                        onChange={(e) => {
                            setEmail(e.target.value);
                            setError("");
                        }}
                        error={error}
                    />

                    <Button type="submit" disabled={loading}>
                        {loading ? "Sending..." : "Send Reset Link"}
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