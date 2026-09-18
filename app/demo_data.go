package app

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Demo personas: realistic, internally consistent records so every screen has something true to show.
// Sales and stock purchases are generated from seedStock (stock.go); this file holds the human details.

type seedPersona struct {
	Phone, Pin, Name, Lang, Role string
	Profile                      map[string]any
	Memories                     []string
	VPA                          string
	Expenses                     []seedExpense // recurring non-stock expenses
	Udhaar                       []seedUdhaar
	Chats                        []seedChat
	Tickets                      []seedTicket
}

type seedExpense struct {
	Note       string
	Rupees     int
	EveryNDays int
}

type seedUdhaar struct {
	Party   string
	Rupees  int
	Given   bool
	DaysAgo int
}

type seedTurn struct {
	User, Assistant string
	Steps           []Step
}

type seedChat struct {
	Title, Stage, Summary string
	DaysAgo               int
	Turns                 []seedTurn
}

type seedTicket struct {
	Subject, Body, Status string
	DaysAgo               int
}

func steps(pairs ...string) []Step {
	out := make([]Step, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, Step{Tool: pairs[i], Label: pairs[i+1], OK: true, Ms: int64(400 + 900*(i/2))})
	}
	return out
}

var personas = []seedPersona{
	{
		Phone: "9000000001", Pin: "1111", Name: "Lakshmi Devi", Lang: "te", Role: "merchant", VPA: "lakshmi.idli@okaxis",
		Profile: map[string]any{
			"owner_name": "Lakshmi Devi", "business_name": "Amma Idli Bandi", "business_type": "Street food cart — idli, dosa, vada, pesarattu",
			"business_domain": "food & street vending", "business_stage": "growing (5+ years)",
			"city": "Vijayawada", "district": "NTR district", "state": "Andhra Pradesh", "pincode": "520002",
			"landmark": "Governorpet bus stop, opposite Kaleswara Rao Market", "shop_hours": "6:00 – 11:00 am, closed Sundays",
			"years_running": 6, "employees": 1, "gender": "female", "social_category": "OBC", "age": 38,
			"upi_vpa": "lakshmi.idli@okaxis", "upi_apps": []string{"PhonePe", "Paytm soundbox"},
			"bank":                  "Union Bank of India, Governorpet branch",
			"documents":             []string{"Aadhaar", "PAN", "Street vendor certificate (VMC)", "Savings account (Union Bank)", "Ration card"},
			"monthly_turnover_band": "₹90,000–₹1,10,000",
			"goal":                  "Buy a second cart near Benz Circle for the office crowd and hire one more helper",
		},
		Memories: []string{
			"Sells idli, dosa, vada and pesarattu from 6 to 11 am near Governorpet bus stop; closed on Sundays.",
			"Husband Ravi grinds the batter at home every night; helper Venkatamma works the tawa.",
			"Lost ₹450 in August to a customer who showed a fake 'payment successful' screen.",
			"Masala dosa sells out by 9:30 am on Saturdays — wants a second tawa.",
			"Afraid of moneylender interest; wants a bank loan under ₹50,000 for a second cart.",
			"Buys rice and dal from Sri Lakshmi Ganapathi Traders every third day.",
		},
		Expenses: []seedExpense{
			{"Helper wages — Venkatamma", 2100, 7},
			{"Chutney vegetables & curry leaves (Rythu Bazaar)", 180, 1},
			{"VMC vending fee", 300, 30},
			{"Mobile recharge (Jio)", 239, 28},
			{"Cart repair — wheel & umbrella", 650, 40},
		},
		Udhaar: []seedUdhaar{
			{"Suresh (auto driver)", 240, true, 4},
			{"Union Bank staff tiffin order", 1800, true, 9},
			{"Union Bank staff tiffin order", 1000, false, 2},
			{"Ravi Teja (SRR college hostel)", 360, true, 3},
			{"Padma akka (flower stall)", 150, true, 1},
		},
		Chats: []seedChat{
			{Title: "నకిలీ పేమెంట్ స్క్రీన్‌షాట్", Stage: "safety", DaysAgo: 6,
				Summary: "Merchant described a customer showing a payment screen without a soundbox alert; Finro matched the fake-screenshot pattern and gave safety steps.",
				Turns: []seedTurn{{
					User:      "ఒక కస్టమర్ ఫోన్‌లో 'పేమెంట్ సక్సెస్' అని చూపించి వెళ్ళిపోయాడు, కానీ సౌండ్‌బాక్స్ ఏమీ చెప్పలేదు. ఇది మోసమా?",
					Assistant: "ఇది **నకిలీ పేమెంట్ స్క్రీన్‌షాట్** మోసం లాగా ఉంది. కస్టమర్ ఫోన్ స్క్రీన్ రుజువు కాదు — మీ సౌండ్‌బాక్స్ లేదా బ్యాంక్ SMS మాత్రమే రుజువు.\n\n1. టిఫిన్ ఇచ్చే ముందు మీ PhonePe హిస్టరీ చూడండి.\n2. రద్దీ సమయంలో సౌండ్‌బాక్స్ వాల్యూమ్ పెంచి ఉంచండి.\n3. డబ్బు పోతే గంటలోపు **1930**కి కాల్ చేయండి.",
					Steps:     steps("check_scam", "Checked against 12 known scam patterns"),
				}},
			},
			{Title: "శనివారం అమ్మకాలు", Stage: "records", DaysAgo: 3,
				Summary: "Merchant logged Saturday sales and a rice purchase; Finro noted masala dosa demand rising.",
				Turns: []seedTurn{
					{
						User:      "శనివారం 52 ప్లేట్ల ఇడ్లీ, 24 మసాలా దోసెలు అమ్మాను. బియ్యం 10 కిలోలు ₹420కి కొన్నాను.",
						Assistant: "ఖాతాలో రాయడానికి సిద్ధం చేశాను — కార్డ్‌లో చూసి **సేవ్** నొక్కండి.\n\nమీ రికార్డుల ప్రకారం **మసాలా దోసె అమ్మకాలు గత మూడు వారాల కంటే సుమారు 20% పెరిగాయి**. శనివారాలకు పిండి కొంచెం ఎక్కువ సిద్ధం చేసుకోండి.",
						Steps:     steps("add_records", "Prepared 3 khata entries for confirmation", "get_records_summary", "Read 30 days of khata records"),
					},
					{
						User:      "మినపప్పు ఎంత ఉంది?",
						Assistant: "**మినపప్పు దాదాపు అయిపోయింది** — ఈ రోజే కొనండి. వారానికి సరిపడా 3.5 కిలోలు కొంటే సరిపోతుంది. కొబ్బరికాయలు కూడా 2 రోజులకే ఉన్నాయి.",
						Steps:     steps("stock_advice", "Analysed item run rates → 3 to buy more, 1 to buy less"),
					},
				},
			},
			{Title: "స్టాండీ కోసం QR", Stage: "onboarding", DaysAgo: 1,
				Summary: "Merchant asked to reprint the shop UPI QR for a new standee.",
				Turns: []seedTurn{{
					User:      "కొత్త స్టాండీ కోసం నా QR మళ్ళీ ప్రింట్ చేయాలి.",
					Assistant: "మీ QR సిద్ధంగా ఉంది — **lakshmi.idli@okaxis**, పేరు *Amma Idli Bandi*. 'స్టాండీ ప్రింట్' నొక్కండి. బండికి అతికించే ముందు ఒకసారి మీరే స్కాన్ చేసి మీ పేరే వస్తుందో చూడండి.",
					Steps:     steps("generate_upi_qr", "Generated UPI QR for lakshmi.idli@okaxis"),
				}},
			},
		},
		Tickets: []seedTicket{
			{"Soundbox announces in Hindi, not Telugu", "My Paytm soundbox speaks Hindi. Can Finro help me change it to Telugu?", "open", 5},
		},
	},
	{
		Phone: "9000000002", Pin: "2222", Name: "Ramesh Gupta", Lang: "hi", Role: "merchant", VPA: "guptakirana@ybl",
		Profile: map[string]any{
			"owner_name": "Ramesh Gupta", "business_name": "Gupta General Store", "business_type": "Kirana / general store with home delivery on WhatsApp",
			"business_domain": "retail / kirana", "business_stage": "growing (5+ years)", "needs": []string{"loan", "subsidy", "market access"},
			"city": "Varanasi", "district": "Varanasi", "state": "Uttar Pradesh", "pincode": "221005",
			"landmark": "Lanka, near BHU main gate", "shop_hours": "7:30 am – 10:00 pm, all days",
			"years_running": 14, "employees": 2, "gender": "male", "social_category": "General", "age": 46,
			"upi_vpa": "guptakirana@ybl", "upi_apps": []string{"PhonePe", "Paytm", "BHIM"},
			"bank":                  "State Bank of India, Lanka branch (current account)",
			"documents":             []string{"Aadhaar", "PAN", "Udyam registration", "GST (composition)", "Current account (SBI)", "Shop & establishment licence", "FSSAI basic registration"},
			"monthly_turnover_band": "₹4–4.5 lakh",
			"goal":                  "Launch a packaged namkeen brand 'Kashi Crunch' and supply 20 shops in Lanka and Assi",
		},
		Memories: []string{
			"Got a call from a fake 'Paytm soundbox KYC' agent asking for OTP in July; did not share it.",
			"Son Aman (22) handles WhatsApp orders and home delivery on a scooty.",
			"Has an unused back room (10×12 ft) that could become a namkeen packing unit.",
			"Buys wholesale from Vishweshwarganj mandi every fourth day by e-rickshaw.",
			"Weekend sales are about a third higher because of BHU students and families.",
			"Interested in PMEGP or MUDRA Kishore for a packing machine (~₹3.5 lakh).",
		},
		Expenses: []seedExpense{
			{"Electricity bill (PVVNL)", 2900, 30},
			{"Staff salary — Raju & Deepak", 8000, 15},
			{"Shop rent", 12000, 30},
			{"E-rickshaw from mandi", 150, 4},
			{"Carry bags & packing material", 320, 10},
			{"Scooty petrol for deliveries", 250, 5},
		},
		Udhaar: []seedUdhaar{
			{"Sharma ji (Lane 3)", 1350, true, 6},
			{"Hotel Annapurna", 4200, true, 12},
			{"Hotel Annapurna", 2000, false, 3},
			{"Pandey Tailors", 640, true, 5},
			{"Mishra ji (hostel warden)", 980, true, 2},
			{"Sharma ji (Lane 3)", 500, false, 1},
		},
		Chats: []seedChat{
			{Title: "साउंडबॉक्स KYC कॉल", Stage: "safety", DaysAgo: 8,
				Summary: "Merchant reported a KYC call asking for OTP; Finro identified the fake soundbox agent pattern.",
				Turns: []seedTurn{{
					User:      "किसी ने फ़ोन करके कहा कि मेरे Paytm साउंडबॉक्स का KYC आज ख़त्म हो रहा है और OTP माँगा।",
					Assistant: "यह **नकली साउंडबॉक्स एजेंट / KYC** वाला धोखा है। कोई भी असली कंपनी कॉल पर OTP नहीं माँगती।\n\n1. कोई OTP न बताएँ।\n2. सर्विस रिक्वेस्ट सिर्फ़ Paytm ऐप से करें।\n3. अपनी दुकान का QR स्कैन करके देखें — नाम *Gupta General Store* ही दिखना चाहिए।",
					Steps:     steps("check_scam", "Checked against 12 known scam patterns"),
				}},
			},
			{Title: "शनिवार का माल", Stage: "records", DaysAgo: 4,
				Summary: "Merchant asked what to stock up for the weekend; Finro used item run rates to suggest cold drinks and Maggi up, Surf Excel down.",
				Turns: []seedTurn{{
					User:      "शनिवार-रविवार के लिए कौन सा सामान ज़्यादा मँगाऊँ?",
					Assistant: "पिछले हफ़्तों का हिसाब देखा: **कोल्ड ड्रिंक की बिक्री करीब 25% बढ़ी है** और शनिवार-रविवार को दुकान की बिक्री लगभग एक-तिहाई ज़्यादा रहती है।\n\n1. कोल्ड ड्रिंक और मैगी का स्टॉक बढ़ाएँ।\n2. Parle-G और अरहर दाल ख़त्म हैं — आज ही मँगाएँ।\n3. Surf Excel करीब 50 दिन का पड़ा है — इस बार न मँगाएँ।",
					Steps:     steps("stock_advice", "Analysed item run rates → 4 to buy more, 1 to buy less"),
				}},
			},
			{Title: "नमकीन यूनिट के लिए लोन", Stage: "funding", DaysAgo: 2,
				Summary: "Merchant asked about loans for a namkeen packing unit; Finro found MUDRA, PMFME and UP ODOP schemes with official links.",
				Turns: []seedTurn{{
					User:      "नमकीन पैकिंग यूनिट के लिए लोन या सब्सिडी कहाँ से मिल सकती है?",
					Assistant: "आपकी प्रोफ़ाइल (उत्तर प्रदेश, किराना, 14 साल) के हिसाब से **PM MUDRA (किशोर)**, **PMFME** और **UP ODOP** योजनाएँ देखने लायक हैं। हर कार्ड में सरकारी लिंक है — वहाँ पूरे नियम ज़रूर पढ़ें।\n\nअगला कदम: आपका ब्लूप्रिंट बनाकर पात्रता जाँचें?",
					Steps:     steps("scheme_intake", "Profile complete for scheme search", "find_schemes", "Matched 4 of 27 schemes to your state, category & business"),
				}},
			},
		},
		Tickets: []seedTicket{
			{"Voice input on old phone", "Mic button does not work on my son's old Redmi phone in Chrome.", "resolved", 9},
		},
	},
	{
		Phone: "9000000000", Pin: "0000", Name: "Mentor Desk", Lang: "en", Role: "admin", VPA: "sairamtea@ybl",
		Profile: map[string]any{
			"owner_name": "Mentor Desk (sample: Srinivas Rao)", "business_name": "Sai Ram Tea Point", "business_type": "Tea and snacks counter near the AP Secretariat",
			"business_domain": "food & street vending", "business_stage": "running (1-5 years)", "needs": []string{"loan", "equipment"},
			"city": "Amaravati (Velagapudi)", "district": "Guntur", "state": "Andhra Pradesh", "pincode": "522503",
			"landmark": "Secretariat Block 2 parking, Velagapudi", "shop_hours": "6:30 am – 8:00 pm, Mon–Sat",
			"years_running": 3, "employees": 2, "gender": "male", "social_category": "OBC", "age": 34,
			"upi_vpa": "sairamtea@ybl", "upi_apps": []string{"PhonePe", "Google Pay"},
			"bank":                  "Andhra Pragathi Grameena Bank, Thullur",
			"documents":             []string{"Aadhaar", "PAN", "Udyam registration", "Savings account (APGB)", "FSSAI basic registration"},
			"monthly_turnover_band": "₹70,000–₹85,000",
			"goal":                  "Add a steel tea-and-snacks kiosk with a small fridge for cold drinks and buttermilk",
		},
		Memories: []string{
			"Morning rush 8:30–10:30 am from Secretariat staff; second peak 4–5 pm.",
			"Samosas come from a supplier in Thullur at ₹7 each, delivered at 9 am.",
			"Counter is closed on Sundays and government holidays because offices are shut.",
			"Wants a fridge so buttermilk and cold drinks can be sold in summer.",
		},
		Expenses: []seedExpense{
			{"Counter space rent (Secretariat canteen society)", 6000, 30},
			{"Helper wages — Babu & Kiran (part-time)", 600, 1},
			{"Samosas from Thullur supplier", 280, 1},
			{"Gas refill (commercial)", 1850, 10},
			{"Snacks supplier delivery charge", 60, 1},
		},
		Udhaar: []seedUdhaar{
			{"Revenue department section tea account", 2400, true, 6},
			{"Revenue department section tea account", 1500, false, 1},
			{"Security guards (Block 2)", 420, true, 3},
		},
		Chats: []seedChat{
			{Title: "This week at the counter", Stage: "records", DaysAgo: 2,
				Summary: "Mentor sample chat: weekly performance and stock check for the tea counter.",
				Turns: []seedTurn{{
					User:      "How did the tea counter do this week, and what should I order?",
					Assistant: "Chai is steady at about **120 cups a day**, and **bun maska is up about 16%** compared with the three weeks before. Milk will last only about 2 days — order before tomorrow's morning rush. Samosa sales dipped a little, so ask the supplier for a smaller delivery after 11 am.",
					Steps:     steps("get_records_summary", "Read 30 days of khata records", "stock_advice", "Analysed item run rates → 3 to buy more, 1 to buy less"),
				}},
			},
		},
		Tickets: []seedTicket{
			{"Mentor note: demo reset", "Use `go run ./cmd/dev reset-demo` to restore sample data before a judging round.", "resolved", 1},
		},
	},
}

// seedChats writes each persona's past conversations with tool trails.
func seedChats(ctx context.Context, tx pgx.Tx, uid int64, p seedPersona) error {
	for _, c := range p.Chats {
		var sid int64
		if err := tx.QueryRow(ctx, `INSERT INTO sessions(user_id, title, stage, summary, created_at, updated_at)
			VALUES($1,$2,$3,$4, now() - make_interval(days => $5), now() - make_interval(days => $5)) RETURNING id`,
			uid, c.Title, c.Stage, c.Summary, c.DaysAgo).Scan(&sid); err != nil {
			return err
		}
		for i, t := range c.Turns {
			mins := 30 - i*3
			if _, err := tx.Exec(ctx, `INSERT INTO messages(session_id, role, content, meta, created_at)
				VALUES($1,'user',$2,'{"source":"text"}', now() - make_interval(days => $3, mins => $4))`, sid, t.User, c.DaysAgo, mins); err != nil {
				return err
			}
			meta := map[string]any{"steps": t.Steps, "stage": c.Stage}
			if _, err := tx.Exec(ctx, `INSERT INTO messages(session_id, role, content, meta, created_at)
				VALUES($1,'assistant',$2,$3, now() - make_interval(days => $4, mins => $5))`, sid, t.Assistant, meta, c.DaysAgo, mins-1); err != nil {
				return err
			}
		}
	}
	return nil
}
