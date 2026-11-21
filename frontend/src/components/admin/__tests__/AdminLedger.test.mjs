import test from "node:test";
import assert from "node:assert";

test("Ledger double-entry balance verification checks total debit equals total credit", () => {
  const isBalanced = (entries) => {
    let totalDebit = 0;
    let totalCredit = 0;
    for (const e of entries) {
      const amt = parseFloat(e.amount) || 0;
      if (e.entry_type === "DEBIT") totalDebit += amt;
      else if (e.entry_type === "CREDIT") totalCredit += amt;
    }
    return Math.abs(totalDebit - totalCredit) < 0.0001 && totalDebit > 0;
  };

  const balancedEntries = [
    { entry_type: "DEBIT", amount: "1000.00" },
    { entry_type: "CREDIT", amount: "900.00" },
    { entry_type: "CREDIT", amount: "100.00" },
  ];
  assert.strictEqual(isBalanced(balancedEntries), true);

  const unbalancedEntries = [
    { entry_type: "DEBIT", amount: "1000.00" },
    { entry_type: "CREDIT", amount: "900.00" },
  ];
  assert.strictEqual(isBalanced(unbalancedEntries), false);

  const emptyEntries = [];
  assert.strictEqual(isBalanced(emptyEntries), false);
});

test("Ledger reversal identification correctly identifies reversed and compensating reversal journals", () => {
  const classifyJournalReversal = (journal) => {
    if (journal.transaction_type === "REVERSAL" && journal.reversal_reference_id) {
      return "COMPENSATING_REVERSAL";
    }
    if (journal.is_reversed && journal.reversed_by_journal_id) {
      return "REVERSED_ORIGINAL";
    }
    return "STANDARD";
  };

  assert.strictEqual(
    classifyJournalReversal({
      transaction_type: "REVERSAL",
      reversal_reference_id: "00000000-0000-0000-0000-000000000001",
    }),
    "COMPENSATING_REVERSAL"
  );

  assert.strictEqual(
    classifyJournalReversal({
      transaction_type: "PAYMENT",
      is_reversed: true,
      reversed_by_journal_id: "00000000-0000-0000-0000-000000000002",
    }),
    "REVERSED_ORIGINAL"
  );

  assert.strictEqual(
    classifyJournalReversal({
      transaction_type: "PAYMENT",
      is_reversed: false,
    }),
    "STANDARD"
  );
});

test("Ledger table search and filter matching behaves consistently", () => {
  const journals = [
    { id: "j1", transaction_type: "PAYMENT", source_id: "pay_100", status: "POSTED", description: "Ticket sale" },
    { id: "j2", transaction_type: "REFUND", source_id: "rfnd_200", status: "POSTED", description: "Customer refund" },
    { id: "j3", transaction_type: "REVERSAL", source_id: "rev_300", status: "POSTED", description: "Compensating journal" },
    { id: "j4", transaction_type: "PAYMENT", source_id: "pay_400", status: "DRAFT", description: "Draft sale" },
  ];

  const filterJournals = (list, { type = "ALL", status = "ALL", search = "" }) => {
    return list.filter((item) => {
      if (type !== "ALL" && item.transaction_type !== type) return false;
      if (status !== "ALL" && item.status !== status) return false;
      if (search) {
        const query = search.toLowerCase();
        const matches =
          item.id.toLowerCase().includes(query) ||
          item.source_id.toLowerCase().includes(query) ||
          item.description.toLowerCase().includes(query);
        if (!matches) return false;
      }
      return true;
    });
  };

  assert.strictEqual(filterJournals(journals, {}).length, 4);
  assert.strictEqual(filterJournals(journals, { type: "REFUND" }).length, 1);
  assert.strictEqual(filterJournals(journals, { status: "DRAFT" }).length, 1);
  assert.strictEqual(filterJournals(journals, { search: "pay" }).length, 2);
  assert.strictEqual(filterJournals(journals, { search: "nonexistent" }).length, 0);
});
