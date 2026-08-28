import { useState, useEffect } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom"; 
import AuthLayout from "../../components/layout/AuthLayout";
import OtpInput from "../../components/ui/OtpInput";
import Button from "../../components/ui/Button";
import { verifyOTP } from "../../api/authApi"; 

function formatTime(s) {
    const m = String(Math.floor(s / 60)).padStart(2, "0");
    const sec = String(s % 60).padStart(2, "0");
    return `${m}:${sec}`;
}

export default function OtpPage() {
    const [code, setCode] = useState("");
    const [error, setError] = useState("");
    const [secondsLeft, setSecondsLeft] = useState(45);

    const location = useLocation(); 
    const navigate = useNavigate(); 

    const email = location.state?.email; 

    useEffect(() => {
        if (!email) {
            navigate("/register", { replace: true });
        }
    }, [email, navigate]);


    const maskedEmail = email
        ? email.replace(
              /^(.{1,2})(.*)(@.*)$/,
              (_, start, middle, domain) =>
                  `${start}${"*".repeat(Math.min(middle.length, 5))}${domain}`
          )
        : "";

    useEffect(() => {
        if (secondsLeft <= 0) return;

        const timer = setInterval(
            () => setSecondsLeft((s) => s - 1),
            1000
        );

        return () => clearInterval(timer);
    }, [secondsLeft]);

    async function handleSubmit(e) { 
        e.preventDefault();

        if (code.length < 6) {
            setError("Enter the full 6-digit code");
            return;
        }

        if (!email) { 
            setError("Email address is missing. Please register again.");
            return;
        }

        setError("");

        try { 
            await verifyOTP({
                email,
                otp: code, 
            });

            navigate("/login", { 
                replace: true,
            });
        } catch (error) { 
            console.error("OTP verification failed:", error); 

            const message =
                error.response?.data?.message ||
                "OTP verification failed. Please try again.";

            setError(message);
        }
    }

    function handleResend() {
        console.log("Would resend code");
        setSecondsLeft(45);
    }

    return (
        <AuthLayout
            title={
                <>
                    Discover.
                    <br />
                    Book.
                    <br />
                    Experience.
                </>
            }
            description="Your events are waiting for you."
        >
            <div className="flex flex-col items-center gap-6 w-full max-w-md">
                <div className="bg-white rounded-lg shadow p-8 w-full flex flex-col gap-8">
                    <div className="flex flex-col items-center gap-1 text-center">
                        <span className="font-display text-xl font-semibold text-primary">
                            EventHub
                        </span>

                        <h2 className="font-display text-2xl font-semibold text-ink pt-2">
                            Verify your email
                        </h2>

                        <p className="text-sm text-ink/60">
                            We've sent a verification code to
                            <br />
                            <span className="font-semibold text-ink">
                                {maskedEmail}
                            </span>
                        </p>
                    </div>

                    <form
                        onSubmit={handleSubmit}
                        className="flex flex-col gap-8"
                    >
                        <div>
                            <OtpInput
                                length={6}
                                value={code}
                                onChange={setCode}
                            />

                            {error && (
                                <p className="text-xs text-red-500 mt-2">
                                    {error}
                                </p>
                            )}
                        </div>

                        <Button type="submit">
                            Verify Email
                        </Button>
                    </form>

                    <div className="flex flex-col gap-4">
                        <p className="text-center text-sm text-ink/60">
                            Didn't receive the code?{" "}
                            {secondsLeft > 0 ? (
                                <span className="text-ink/40">
                                    Resend code in {formatTime(secondsLeft)}
                                </span>
                            ) : (
                                <button
                                    onClick={handleResend}
                                    className="text-primary font-semibold"
                                >
                                    Resend code
                                </button>
                            )}
                        </p>

                        <div className="border-t border-border pt-4 flex justify-center">
                            <Link
                                to="/register"
                                className="flex items-center gap-1 text-sm text-primary"
                            >
                                <svg
                                    width="12"
                                    height="12"
                                    viewBox="0 0 12 12"
                                    fill="none"
                                >
                                    <path
                                        d="M1.33333 10.6667H2.28333L8.8 4.15L7.85 3.2L1.33333 9.71667V10.6667V10.6667M0 12V9.16667L8.8 0.383333C8.93333 0.261111 9.08056 0.166667 9.24167 0.1C9.40278 0.0333333 9.57222 0 9.75 0C9.92778 0 10.1 0.0333333 10.2667 0.1C10.4333 0.166667 10.5778 0.266667 10.7 0.4L11.6167 1.33333C11.75 1.45556 11.8472 1.6 11.9083 1.76667C11.9694 1.93333 12 2.1 12 2.26667C12 2.44444 11.9694 2.61389 11.9083 2.775C11.8472 2.93611 11.75 3.08333 11.6167 3.21667L2.83333 12H0V12"
                                        fill="currentColor"
                                    />
                                </svg>

                                Wrong email address? Change email
                            </Link>
                        </div>
                    </div>
                </div>

                <p className="text-xs text-ink/50">
                    © 2026 EventHub Inc.
                </p>
            </div>
        </AuthLayout>
    );
}