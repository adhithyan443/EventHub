import useEventCreationStore from "../../store/eventCreationStore";

/**
 * Reusable Payout Bank Account Card.
 * Renders the event-level linked business bank account configuration
 * for both General Admission and Reserved Seating ticketing flows.
 */
export default function PayoutBankAccountCard({ className = "" }) {
  const payoutAccount = useEventCreationStore((state) => state.payoutAccount);

  if (!payoutAccount) return null;

  return (
    <div
      className={`bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col ${className}`}
    >
      <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
        <h2 className="text-base font-bold text-[#141b2b]">Payout Bank Account</h2>
        <p className="text-xs text-[#565e74] mt-1">
          Ticket sale revenues will be deposited into this linked business account.
        </p>
      </div>

      <div className="p-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="size-12 rounded-xl bg-[#00685f]/10 border border-[#00685f]/20 flex items-center justify-center text-[#00685f] text-xl font-bold select-none">
            🏦
          </div>
          <div className="flex flex-col">
            <span className="font-bold text-sm text-[#141b2b]">
              {payoutAccount.bankName}
            </span>
            <span className="text-xs text-[#565e74]">
              {payoutAccount.accountHolder} &bull; {payoutAccount.accountNumberMasked}
            </span>
          </div>
        </div>

        <span className="text-xs font-semibold text-[#00685f] bg-[#00685f]/10 px-3 py-1.5 rounded-full">
          Primary Account
        </span>
      </div>
    </div>
  );
}
