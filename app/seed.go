package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var scamCatalogue = []struct {
	Slug, Title, Desc, Source string
	Flags, Actions            []string
}{
	{"collect-request", "“Accept to receive money” collect request",
		"Scammer sends a UPI collect request saying you will RECEIVE money. Entering your PIN actually SENDS money.",
		"https://www.npci.org.in/what-we-do/upi/upi-safety-awareness",
		[]string{"Request says 'receive' but asks for UPI PIN", "Unknown sender, urgent tone", "Often after an OLX/Quikr style buyer chat"},
		[]string{"You never need a PIN to receive money", "Decline the request in your UPI app", "Report the UPI ID inside the app"}},
	{"fake-screenshot", "Fake payment screenshot",
		"Customer shows an edited 'payment successful' screen or sends a screenshot, but no money reached your account.",
		"https://cybercrime.gov.in/",
		[]string{"Customer shows phone quickly and leaves", "No soundbox announcement or bank SMS", "Screenshot sent on WhatsApp instead of real payment"},
		[]string{"Hand over goods only after soundbox/SMS confirms", "Check the UPI app history, not the customer's screen", "Keep a soundbox or notification ON"}},
	{"qr-to-receive", "“Scan this QR to get your refund”",
		"Scammer sends a QR code claiming scanning it gives you money. Scanning a QR is only ever for paying.",
		"https://www.rbi.org.in/commonman/English/Scripts/BeAware.aspx",
		[]string{"Someone asks YOU to scan to receive", "Refund / prize / buyer advance offered", "Amount pre-filled"},
		[]string{"Never scan a QR to receive money", "Ask them to send to your UPI ID instead", "Block and report the number"}},
	{"screen-share", "Screen-sharing / remote app trap",
		"Fake support agent asks you to install AnyDesk, TeamViewer or a 'KYC app' and then watches your screen and OTPs.",
		"https://www.rbi.org.in/commonman/English/Scripts/BeAware.aspx",
		[]string{"Asked to install an app from a link", "Caller says 'I'll fix it for you'", "Asks you to read out an OTP"},
		[]string{"Never install apps on a stranger's instructions", "Uninstall remote apps you did not choose", "Call your bank's number printed on the card"}},
	{"kyc-expiry", "KYC / account-block SMS",
		"SMS says your bank, Paytm or soundbox KYC expires today and your account will be blocked; link steals login details.",
		"https://sachet.rbi.org.in/",
		[]string{"Threat of block within hours", "Short link or unknown number", "Asks for Aadhaar, PAN, OTP"},
		[]string{"Banks never block accounts by SMS link", "Visit the branch or official app", "Forward the SMS to 1909 / report on Sanchar Saathi Chakshu"}},
	{"fake-customer-care", "Fake customer-care number",
		"Searching Google for a helpline shows a scam number. The 'agent' asks for PIN or to approve a request.",
		"https://cybercrime.gov.in/",
		[]string{"Number found on search results or social media", "Agent asks for PIN/OTP", "Refund needs a 'small verification payment'"},
		[]string{"Use the help section inside the official app", "No real agent asks for your PIN", "Call 1930 if money is lost"}},
	{"overpayment-refund", "Overpayment “please return the extra”",
		"Scammer claims they sent ₹5,000 instead of ₹500 by mistake and asks you to return the difference — nothing was sent.",
		"https://cybercrime.gov.in/",
		[]string{"Pressure to refund quickly", "Fake credit SMS from a normal mobile number", "Bank balance did not actually change"},
		[]string{"Check your bank balance in the official app", "Real bank SMS come from sender IDs like 'AX-SBIINB'", "Let the bank reverse genuine mistakes"}},
	{"electricity-bill", "Electricity disconnection threat",
		"SMS/WhatsApp says power will be cut tonight unless you call an 'officer' and pay a pending bill.",
		"https://www.pib.gov.in/factcheck.aspx",
		[]string{"Personal mobile number of 'officer'", "Deadline of 'tonight 9:30pm'", "Asks to download app or pay small amount"},
		[]string{"DISCOMs do not send such messages from mobile numbers", "Pay only on official DISCOM app/website", "Report on Chakshu (sancharsaathi.gov.in)"}},
	{"task-job", "Part-time task / like-and-earn job",
		"Offers easy earnings for liking videos or reviews. Small first payouts, then asks you to 'invest' for bigger tasks.",
		"https://www.mha.gov.in/en/commonpages/cyber-crime",
		[]string{"Telegram/WhatsApp groups", "Pay-to-unlock tasks", "Returns that look too good"},
		[]string{"Real jobs never ask you to pay", "Leave and report the group", "Report at cybercrime.gov.in"}},
	{"soundbox-fake", "Fake soundbox agent",
		"A person claims to be from Paytm/PhonePe to 'upgrade' your soundbox and asks for OTP or to link a new QR.",
		"https://cybercrime.gov.in/",
		[]string{"Unscheduled visit or call", "Wants your phone to 'update' the device", "Replaces your QR sticker"},
		[]string{"Verify via the official app's service request", "Never share OTP", "Check that your printed QR shows YOUR name when scanned"}},
	{"qr-sticker-swap", "QR sticker swap at the counter",
		"Someone pastes their own QR over yours so customer payments go to the scammer.",
		"https://www.npci.org.in/what-we-do/upi/upi-safety-awareness",
		[]string{"Sticker looks slightly different or raised", "Customers say paid but no alert", "Name shown on payment is not yours"},
		[]string{"Scan your own QR every morning — name must be yours", "Laminate and keep QR behind the counter glass", "Use a soundbox so every payment is announced"}},
	{"digital-arrest", "“Digital arrest” police/courier call",
		"Callers pretending to be police, CBI or courier officials say a parcel in your name has drugs and demand money to 'clear' you.",
		"https://www.pib.gov.in/factcheck.aspx",
		[]string{"Video call with uniform/background of police station", "Told not to tell family", "Asked to transfer money for 'verification'"},
		[]string{"Police never arrest on video calls", "Hang up and tell family", "Call 1930 immediately"}},
}

var seedFAQs = []struct{ Topic, Q, A string }{
	{"safety", "Do I need to enter my UPI PIN to receive money?", "No. Your UPI PIN is only for sending money. If any app, QR or person asks for the PIN to 'receive', it is a scam."},
	{"safety", "I lost money to a UPI scam. What do I do right now?", "Call 1930 (National Cyber Crime Helpline) immediately and report at cybercrime.gov.in. Also inform your bank from the official app. Reporting within the first hour gives the best chance of freezing the money."},
	{"safety", "How does Finro check if a message is a scam?", "Finro compares the message or screenshot against known UPI scam patterns collected from RBI, NPCI and cybercrime advisories plus recent news, and explains exactly which warning signs matched."},
	{"qr", "Is the QR code Finro makes a real UPI QR?", "Yes. It is a standard UPI QR (upi://pay link) with your UPI ID and shop name. Any UPI app — PhonePe, Google Pay, Paytm, BHIM — can pay you with it. Finro never receives or holds your money."},
	{"qr", "Can I put a fixed amount on the QR?", "Yes. For a fixed-price item you can add an amount. For a shop counter, leave the amount empty so customers enter it."},
	{"records", "Is my Khata (records) data private?", "Your records are stored in your account only and used to prepare your business plan and applications. You can delete your account and data from Settings at any time."},
	{"records", "Can I add entries by speaking?", "Yes. Tap the mic and say something like 'sold 40 idlis for 800 rupees, bought rice for 300'. Finro writes the entries and asks you to confirm before saving."},
	{"funding", "Where does the grant and scheme information come from?", "Finro reads official sources directly — government scheme portals and funding agency call pages — and shows when each source was last checked. Deadlines are shown only when the official page states them."},
	{"funding", "Why does Finro say 'deadline not stated'?", "Some schemes are open all year or the official page does not list a date. Finro never guesses a deadline — always confirm on the official link before applying."},
	{"funding", "Does a good match mean I will get the grant?", "No. The match score explains which eligibility rules you seem to meet, fail, or still need to confirm. The final decision is made by the scheme authority."},
	{"account", "I forgot my PIN. How do I reset it?", "For now, raise a ticket from the Help desk with your registered number and our team will help. You can change your PIN any time from Settings while logged in."},
	{"account", "Which languages does Finro speak?", "English, Hindi, Telugu and Tamil. Change the language from the top bar or Settings; Finro's replies and read-aloud follow your choice."},
}

// Seed inserts demo personas and reference data. Idempotent: existing personas are left untouched.
func Seed(ctx context.Context) error {
	db, err := DB()
	if err != nil {
		return err
	}
	for i, c := range scamCatalogue {
		if _, err := db.Exec(ctx, `INSERT INTO scam_patterns(slug, kind, title, description, red_flags, actions, source_url, created_at)
			VALUES($1,'catalogue',$2,$3,$4,$5,$6, now() - make_interval(mins => $7)) ON CONFLICT (slug) DO NOTHING`,
			c.Slug, c.Title, c.Desc, c.Flags, c.Actions, c.Source, i); err != nil {
			return fmt.Errorf("seed patterns: %w", err)
		}
	}
	if err := seedDirectory(ctx); err != nil {
		return fmt.Errorf("seed directory: %w", err)
	}
	for i, f := range seedFAQs {
		if _, err := db.Exec(ctx, `INSERT INTO faqs(topic, q, a, sort) VALUES($1,$2,$3,$4) ON CONFLICT (q) DO NOTHING`, f.Topic, f.Q, f.A, i); err != nil {
			return fmt.Errorf("seed faqs: %w", err)
		}
	}
	for _, p := range personas {
		if err := seedPersonaData(ctx, p); err != nil {
			return fmt.Errorf("seed %s: %w", p.Name, err)
		}
	}
	return nil
}

func seedPersonaData(ctx context.Context, p seedPersona) error {
	db, _ := DB()
	hash, err := bcrypt.GenerateFromPassword([]byte(p.Pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var uid int64
	err = tx.QueryRow(ctx, `INSERT INTO users(phone, pin_hash, name, lang, role) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (phone) DO NOTHING RETURNING id`, p.Phone, string(hash), p.Name, p.Lang, p.Role).Scan(&uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // already seeded
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO profiles(user_id, data) VALUES($1,$2)`, uid, p.Profile); err != nil {
		return err
	}
	for _, m := range p.Memories {
		if _, err := tx.Exec(ctx, `INSERT INTO memories(user_id, fact) VALUES($1,$2)`, uid, m); err != nil {
			return err
		}
	}
	if p.VPA != "" {
		link, _ := upiLink(p.VPA, p.Profile["business_name"].(string), 0)
		if _, err := tx.Exec(ctx, `INSERT INTO qr_codes(user_id, vpa, payee, upi_link) VALUES($1,$2,$3,$4)`, uid, p.VPA, p.Profile["business_name"], link); err != nil {
			return err
		}
	}
	// 45 days of item-level sales, stock use and restocks; day-sales in the ledger are the sum of item sales.
	if err := seedStockHistory(ctx, tx, uid, p.Phone); err != nil {
		return fmt.Errorf("stock: %w", err)
	}
	today := time.Now()
	for d := 45; d >= 1; d-- {
		day := today.AddDate(0, 0, -d)
		closed := seedStock[p.Phone].ClosedSunday && day.Weekday() == time.Sunday
		for _, e := range p.Expenses {
			if d%e.EveryNDays == 0 && !(closed && e.EveryNDays == 1) {
				if _, err := tx.Exec(ctx, `INSERT INTO ledger_entries(user_id, entry_date, kind, amount_paise, note, source) VALUES($1,$2,'expense',$3,$4,'text')`,
					uid, day, e.Rupees*100, e.Note); err != nil {
					return err
				}
			}
		}
	}
	for _, u := range p.Udhaar {
		kind := "udhaar_received"
		if u.Given {
			kind = "udhaar_given"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO ledger_entries(user_id, entry_date, kind, amount_paise, note, party, source) VALUES($1,$2,$3,$4,'Credit',$5,'voice')`,
			uid, today.AddDate(0, 0, -u.DaysAgo), kind, u.Rupees*100, u.Party); err != nil {
			return err
		}
	}
	if err := seedChats(ctx, tx, uid, p); err != nil {
		return err
	}
	for _, tk := range p.Tickets {
		if _, err := tx.Exec(ctx, `INSERT INTO tickets(user_id, subject, body, status, created_at) VALUES($1,$2,$3,$4, now() - make_interval(days => $5))`, uid, tk.Subject, tk.Body, tk.Status, tk.DaysAgo); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// handleSetup runs migrations + seed. Protected by CRON_SECRET.
func handleSetup(w http.ResponseWriter, r *http.Request) error {
	if s := os.Getenv("CRON_SECRET"); s == "" || r.Header.Get("Authorization") != "Bearer "+s {
		return httpErr(401, "unauthorized")
	}
	if err := Migrate(r.Context()); err != nil {
		return err
	}
	if err := Seed(r.Context()); err != nil {
		return err
	}
	return writeJSON(w, map[string]bool{"ok": true})
}

// ResetDemo deletes the demo personas (cascading their chats, khata, drafts) and seeds them fresh.
func ResetDemo(ctx context.Context) error {
	db, err := DB()
	if err != nil {
		return err
	}
	phones := make([]string, len(personas))
	for i, p := range personas {
		phones[i] = p.Phone
	}
	if _, err := db.Exec(ctx, `DELETE FROM users WHERE phone = ANY($1)`, phones); err != nil {
		return err
	}
	return Seed(ctx)
}
