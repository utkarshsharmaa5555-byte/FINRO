package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var stages = []string{"safety", "onboarding", "records", "blueprint", "funding", "apply"}

type Step struct {
	Tool  string `json:"tool"`
	Label string `json:"label"`
	OK    bool   `json:"ok"`
	Ms    int64  `json:"ms"`
}

type Card struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Nudge struct {
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
	Stage  string `json:"stage"`
}

type turnState struct {
	uid   int64
	lang  string
	image string
	stage string
	steps []Step
	cards []Card
	nudge *Nudge
}

func fn(name, desc string, props map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "function", "function": map[string]any{
		"name": name, "description": desc,
		"parameters": map[string]any{"type": "object", "properties": props, "required": required},
	}}
}

var agentTools = []map[string]any{
	fn("check_scam", "Check a pasted SMS/WhatsApp/call description for UPI or payment fraud. Use whenever the user shares a suspicious message or asks if something is safe.",
		map[string]any{"text": str("the exact message or situation")}, "text"),
	fn("check_scam_screenshot", "Check the image the user attached in this turn (payment screenshot, SMS screenshot, QR) for fraud.",
		map[string]any{"context": str("what the user said about it")}),
	fn("list_scam_patterns", "Get the current catalogue of UPI scam patterns plus scams reported in the news this fortnight.", map[string]any{}),
	fn("generate_upi_qr", "Create a real UPI payment QR for the merchant's shop. Requires a fixed amount > 0. Ask for their UPI ID or amount if unknown.",
		map[string]any{"vpa": str("UPI ID like name@okaxis"), "payee": str("shop name shown to customers"), "amount_rupees": map[string]any{"type": "number", "description": "fixed amount in rupees, must be greater than 0"}}, "vpa", "payee", "amount_rupees"),
	fn("add_records", "Turn what the user sold/spent/gave on credit into khata ledger entries for them to confirm. Use when user mentions sales, expenses or udhaar.",
		map[string]any{"text": str("the user's words about money in/out")}, "text"),
	fn("get_records_summary", "Summarise the merchant's khata: sales, expenses, profit, udhaar.",
		map[string]any{"days": integer("7, 30 or 90")}),
	fn("remember_fact", "Save a durable fact about the merchant or business (family help, goals, past fraud, location, plans). Do not store OTPs, PINs, Aadhaar or account numbers.",
		map[string]any{"fact": str("one short sentence")}, "fact"),
	fn("update_profile", "Update structured business profile fields (business_name, business_type, city, state, years_running, employees, gender, social_category, age, upi_vpa, documents[], goal, monthly_turnover_band).",
		map[string]any{"fields": map[string]any{"type": "object", "description": "fields to set"}}, "fields"),
	fn("stock_advice", "Analyse item-level daily sales, run rates and stock left to tell the merchant what to buy more of and less of this week. Use when the user asks about stock, purchases, what is selling, or what to order.", map[string]any{}),
	fn("build_blueprint", "Generate/refresh the business blueprint from profile, memory and khata records. Use when user wants to grow, needs a loan/grant, or asks for a business plan.",
		map[string]any{"notes": str("new plans or details from this conversation")}),
	fn("scheme_intake", "FIRST step whenever the user asks about government schemes, loans, subsidies or support: checks which personal details (state, gender, age, social category, business domain, stage, needs) are missing and shows a tap-to-fill form.", map[string]any{}),
	fn("find_schemes", "Recommend government schemes personalised to the user's state, gender, age, category, domain, stage and needs, with official links. Only call after scheme_intake reports nothing missing.",
		map[string]any{"focus": str("what the user wants help with, e.g. 'loan for a second cart'")}),
	fn("search_funding", "List live government funding schemes/grants scraped from official sources with freshness info.", map[string]any{}),
	fn("match_funding", "Score live funding schemes against the merchant's latest blueprint with rule-by-rule eligibility.", map[string]any{}),
	fn("draft_application", "Draft an application for one funding scheme (by grant_id from match/search results) pre-filled from the blueprint.",
		map[string]any{"grant_id": integer("")}, "grant_id"),
	fn("set_stage", "Move the session to the journey stage the conversation is now in.",
		map[string]any{"stage": enum(stages...)}, "stage"),
	fn("suggest_next_step", "Offer ONE proactive next step as a tappable chip, e.g. after a scam check suggest making a safe QR; after sales talk suggest a blueprint; after a blueprint suggest matching funding.",
		map[string]any{"label": str("short chip text in the user's language"), "prompt": str("what the user would say if they tap it, in their language"), "stage": enum(stages...)}, "label", "prompt", "stage"),
}

func systemPrompt(ctx context.Context, u *User, sess *chatSession) string {
	bctx, _, _ := businessContext(ctx, u.ID)
	var bp string
	if b, _ := latestBlueprint(ctx, u.ID); b != nil {
		bp = fmt.Sprintf("A blueprint exists (id %d, %s): %v", b.ID, b.CreatedAt, b.Data["one_liner"])
	} else {
		bp = "No blueprint yet."
	}
	return strings.Join([]string{
		"You are Finro — a trusted, warm neighbourhood accountant and financial guide for Indian street vendors, kirana owners and first-time founders.",
		"Journey stages: safety (fraud) → onboarding (UPI QR) → records (khata) → blueprint → funding (live schemes + matching) → apply (drafts). Move the user naturally along it in ONE conversation, never forcing.",
		"Reply ONLY in " + langNames[u.Lang] + " (use its native script), short sentences, max ~120 words, simple words, ₹ amounts. Use **bold** for key warnings and numbered steps for actions.",
		"Always use tools for facts: scam checks, QR, records, blueprint, funding. The app renders tool results as cards, so do not repeat card details at length.",
		"CRITICAL: never write any grant deadline, amount or eligibility from memory. Only refer to schemes returned by tools, by name, and tell the user the card shows the verified details and source.",
		"Never ask for or store OTP, UPI PIN, passwords, full Aadhaar or bank account numbers. If money was lost, tell them to call 1930 immediately.",
		"SCOPE — you are a work assistant, not a general chatbot. You only handle: payment fraud and safety, UPI/QR and getting paid, khata records, stock and buying decisions, business planning, government schemes, funding and applications, and using this app. " +
			"If the user asks about anything else (relationships, health or medicine, food choices, religion, politics, sports, entertainment, studies, coding, general knowledge, or personal life decisions), do NOT answer it even briefly: reply in ONE warm sentence that this is outside Finro's work, then offer the nearest business help and call suggest_next_step. " +
			"Never give medical, legal, investment or relationship advice. If the question is about their business in any way (family members working in the shop, a festival affecting sales, a supplier dispute), it IS in scope — help with it.",
		"SCHEME FLOW: when the user asks about government schemes, loans, subsidies, grants or any support, call scheme_intake first. If details are missing, ask for them warmly (the card lets them tap answers); if the user states them in words, save with update_profile using keys state, gender, age, social_category, business_domain, business_stage, needs (array), then call scheme_intake again. When nothing is missing call find_schemes, then encourage them to read each scheme on its official site.",
		"When the user shares personal/business facts, call remember_fact or update_profile. End most turns with suggest_next_step to proactively guide the journey.",
		"Current stage: " + sess.Stage + ". Session summary so far: " + orNone(sess.Summary),
		"Today: " + time.Now().In(ist).Format("2 Jan 2006") + ". User name: " + u.Name + ".",
		bp,
		"What Finro knows (profile, remembered facts, verified khata numbers):\n" + bctx,
	}, "\n")
}

func orNone(s string) string {
	if s == "" {
		return "(new conversation)"
	}
	return s
}

func (t *turnState) step(tool, label string, ok bool, start time.Time) {
	t.steps = append(t.steps, Step{Tool: tool, Label: label, OK: ok, Ms: time.Since(start).Milliseconds()})
}

// runTool executes one tool call and returns the JSON result fed back to the model.
func runTool(ctx context.Context, t *turnState, call ToolCall) string {
	start := time.Now()
	var args map[string]any
	json.Unmarshal([]byte(call.Function.Arguments), &args)
	s := func(k string) string { v, _ := args[k].(string); return v }
	fail := func(label string, err error) string {
		t.step(call.Function.Name, label, false, start)
		var ae *apiError
		if errors.As(err, &ae) {
			return mustJSON(map[string]string{"error": ae.Msg})
		}
		return mustJSON(map[string]string{"error": "tool failed, apologise briefly and offer another way"})
	}
	switch call.Function.Name {
	case "check_scam", "check_scam_screenshot":
		img := ""
		text := s("text")
		if call.Function.Name == "check_scam_screenshot" {
			if t.image == "" {
				return mustJSON(map[string]string{"error": "no image attached this turn; ask the user to attach the screenshot"})
			}
			img, text = t.image, s("context")
		}
		v, err := checkScam(ctx, t.uid, text, img, t.lang)
		if err != nil {
			return fail("Scam check failed", err)
		}
		n, _ := loadPatterns(ctx, "catalogue", 50)
		t.step(call.Function.Name, fmt.Sprintf("Checked against %d known scam patterns", len(n)), true, start)
		t.cards = append(t.cards, Card{"scam_verdict", v})
		return mustJSON(v)
	case "list_scam_patterns":
		cat, err := loadPatterns(ctx, "catalogue", 50)
		if err != nil {
			return fail("Could not load scam patterns", err)
		}
		news, _ := loadPatterns(ctx, "news", 6)
		t.step("list_scam_patterns", fmt.Sprintf("Loaded %d patterns + %d recent news reports", len(cat), len(news)), true, start)
		t.cards = append(t.cards, Card{"scam_patterns", map[string]any{"catalogue": cat[:min(len(cat), 6)], "news": news}})
		brief := make([]string, 0, len(cat)+len(news))
		for _, p := range cat {
			brief = append(brief, p.Title+": "+p.Description)
		}
		for _, p := range news {
			brief = append(brief, "NEWS "+p.Title)
		}
		return mustJSON(brief)
	case "generate_upi_qr":
		amt, _ := args["amount_rupees"].(float64)
		if amt <= 0 {
			return fail("amount required", errors.New("please specify an amount greater than ₹0 for the QR code"))
		}
		q, err := createQR(ctx, t.uid, s("vpa"), s("payee"), int64(amt*100+0.5))
		if err != nil {
			return fail("QR not created", err)
		}
		t.step("generate_upi_qr", "Generated UPI QR for "+q.VPA, true, start)
		t.cards = append(t.cards, Card{"qr", q})
		return mustJSON(map[string]any{"ok": true, "qr_id": q.ID, "upi_link": q.Link, "tip": "Tell them to print the standee and scan it once to confirm their own name shows."})
	case "add_records":
		res, err := parseEntries(ctx, t.uid, s("text"), "", "text")
		if err != nil {
			return fail("Could not read the entries", err)
		}
		n := len(res["entries"].([]LedgerEntry))
		t.step("add_records", fmt.Sprintf("Prepared %d khata entries for confirmation", n), true, start)
		t.cards = append(t.cards, Card{"records_proposal", res})
		return mustJSON(map[string]any{"proposed": res, "note": "Entries are NOT saved until the user taps Save on the card."})
	case "get_records_summary":
		days := 30
		if d, ok := args["days"].(float64); ok && d > 0 && d <= 365 {
			days = int(d)
		}
		sum, err := recordsSummary(ctx, t.uid, days)
		if err != nil {
			return fail("Could not read khata", err)
		}
		t.step("get_records_summary", fmt.Sprintf("Read %d days of khata records", days), true, start)
		t.cards = append(t.cards, Card{"records_summary", sum})
		sum.Daily = nil
		return mustJSON(sum)
	case "remember_fact":
		if err := addMemory(ctx, t.uid, s("fact")); err != nil {
			return fail("Memory not saved", err)
		}
		t.step("remember_fact", "Remembered: "+truncate(s("fact"), 80), true, start)
		return `{"ok":true}`
	case "update_profile":
		fields, _ := args["fields"].(map[string]any)
		if len(fields) == 0 || len(fields) > 20 {
			return fail("Profile not updated", httpErr(400, "no fields"))
		}
		if _, err := mergeProfile(ctx, t.uid, fields); err != nil {
			return fail("Profile not updated", err)
		}
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		t.step("update_profile", "Updated business profile: "+strings.Join(keys, ", "), true, start)
		return `{"ok":true}`
	case "stock_advice":
		a, err := stockAdvice(ctx, t.uid, t.lang)
		if err != nil {
			return fail("Stock analysis failed", err)
		}
		t.step("stock_advice", fmt.Sprintf("Analysed item run rates → %d to buy more, %d to buy less", len(a.BuyMore), len(a.BuyLess)), true, start)
		t.cards = append(t.cards, Card{"stock_advice", a})
		return mustJSON(map[string]any{"headline": a.Headline, "buy_more": a.BuyMore, "buy_less": a.BuyLess, "note": "The card shows quantities; summarise briefly."})
	case "build_blueprint":
		b, err := buildBlueprint(ctx, t.uid, s("notes"))
		if err != nil {
			return fail("Blueprint failed", err)
		}
		t.step("build_blueprint", "Built blueprint from profile, memory and 30 days of khata", true, start)
		t.cards = append(t.cards, Card{"blueprint", b})
		return mustJSON(map[string]any{"blueprint_id": b.ID, "one_liner": b.Data["one_liner"], "funding_need": b.Data["funding_need"], "missing_info": b.Data["missing_info"]})
	case "scheme_intake":
		profile, _, err := loadProfile(ctx, t.uid)
		if err != nil {
			return fail("Could not read profile", err)
		}
		miss := missingIntake(profile)
		if len(miss) == 0 {
			t.step("scheme_intake", "Profile complete for scheme search", true, start)
			return mustJSON(map[string]any{"missing": []string{}, "known": intakeSummary(profile), "next": "call find_schemes now"})
		}
		t.step("scheme_intake", fmt.Sprintf("Need %d details to personalise schemes", len(miss)), true, start)
		t.cards = append(t.cards, Card{"scheme_intake", map[string]any{"missing": miss, "profile": profile, "options": intakeOptions}})
		return mustJSON(map[string]any{"missing": miss, "note": "A form card is shown. Briefly ask the user to tap their answers in the card (or say them). Do not call find_schemes yet."})
	case "find_schemes":
		res, err := findSchemes(ctx, t.uid, s("focus"), t.lang)
		if err != nil {
			return fail("Scheme search failed", err)
		}
		t.step("find_schemes", fmt.Sprintf("Matched %d of %d schemes to your state, category & business", len(res.Recommendations), res.Considered), true, start)
		t.cards = append(t.cards, Card{"scheme_recs", res})
		brief := make([]map[string]any, len(res.Recommendations))
		for i, r := range res.Recommendations {
			brief[i] = map[string]any{"grant_id": r.GrantID, "title": r.Grant.Title, "fit": r.Fit}
		}
		return mustJSON(map[string]any{"recommended": brief, "suggestions": res.Suggestions, "note": "Cards show summaries and official links. Tell the user to open each official link and read the full rules before applying; offer to build a blueprint and draft an application."})
	case "search_funding":
		gs, err := listGrants(ctx)
		if err != nil {
			return fail("Could not load funding", err)
		}
		health, _ := sourceHealth(ctx)
		open := []Grant{}
		for _, g := range gs {
			if g.Status != "closed" {
				open = append(open, g)
			}
		}
		t.step("search_funding", fmt.Sprintf("Read %d live schemes from %d official sources", len(open), len(health)), true, start)
		t.cards = append(t.cards, Card{"funding_list", map[string]any{"grants": open, "sources": health}})
		brief := make([]map[string]any, len(open))
		for i, g := range open {
			brief[i] = map[string]any{"grant_id": g.ID, "title": g.Title, "source": g.Source}
		}
		return mustJSON(brief)
	case "match_funding":
		b, err := latestBlueprint(ctx, t.uid)
		if err == nil && b == nil {
			b, err = buildBlueprint(ctx, t.uid, "")
		}
		if err != nil {
			return fail("Matching failed", err)
		}
		fits, err := scoreFits(ctx, t.uid, b)
		if err != nil {
			return fail("Matching failed", err)
		}
		strong := 0
		brief := []map[string]any{}
		for _, f := range fits {
			if f.Verdict != "not_a_fit" {
				strong++
				brief = append(brief, map[string]any{"grant_id": f.GrantID, "title": f.Grant.Title, "score": f.Score, "verdict": f.Verdict, "why": f.Why})
			}
		}
		t.step("match_funding", fmt.Sprintf("Checked eligibility rule-by-rule for %d schemes → %d fit", len(fits), strong), true, start)
		t.cards = append(t.cards, Card{"fits", fits})
		return mustJSON(brief)
	case "draft_application":
		id, _ := args["grant_id"].(float64)
		d, err := createDraft(ctx, t.uid, int64(id))
		if err != nil {
			return fail("Draft failed", err)
		}
		t.step("draft_application", fmt.Sprintf("Drafted application (%v gaps marked NEEDED)", d["needed_count"]), true, start)
		t.cards = append(t.cards, Card{"draft", d})
		return mustJSON(map[string]any{"draft_id": d["id"], "needed_count": d["needed_count"], "documents": d["documents"]})
	case "set_stage":
		st := s("stage")
		if !slicesContains(stages, st) {
			return `{"error":"unknown stage"}`
		}
		t.stage = st
		return `{"ok":true}`
	case "suggest_next_step":
		if st := s("stage"); slicesContains(stages, st) {
			t.nudge = &Nudge{Label: s("label"), Prompt: s("prompt"), Stage: st}
		}
		return `{"ok":true}`
	}
	return `{"error":"unknown tool"}`
}

// defaultNudges map the last tool used to the next journey step, so every turn offers a way forward
// even when the model forgets suggest_next_step. Index: 0 label, 1 prompt.
var defaultNudges = map[string]map[string][2]string{
	"onboarding": {
		"en": {"Make a safe UPI QR for my shop", "Make a UPI QR for my shop"},
		"hi": {"मेरी दुकान का सुरक्षित UPI QR बनाओ", "मेरी दुकान का UPI QR बनाओ"},
		"te": {"నా షాప్‌కు సురక్షిత UPI QR చేయండి", "నా షాప్‌కు UPI QR చేయండి"},
		"ta": {"என் கடைக்கு பாதுகாப்பான UPI QR செய்", "என் கடைக்கு UPI QR செய்யுங்கள்"},
	},
	"records": {
		"en": {"Add today's sales to my khata", "Show my khata summary for the last 30 days"},
		"hi": {"आज की बिक्री खाते में जोड़ें", "पिछले 30 दिन का मेरा खाता दिखाओ"},
		"te": {"ఈరోజు అమ్మకాలు ఖాతాలో రాయండి", "గత 30 రోజుల నా ఖాతా సారాంశం చూపించండి"},
		"ta": {"இன்றைய விற்பனையைக் கணக்கில் சேர்", "கடந்த 30 நாள் கணக்குச் சுருக்கம் காட்டு"},
	},
	"blueprint": {
		"en": {"Turn my khata into a business plan", "Build my business blueprint"},
		"hi": {"मेरे खाते से व्यापार योजना बनाओ", "मेरा व्यापार ब्लूप्रिंट बनाओ"},
		"te": {"నా ఖాతాతో వ్యాపార ప్రణాళిక చేయండి", "నా వ్యాపార బ్లూప్రింట్ చేయండి"},
		"ta": {"என் கணக்கிலிருந்து வணிகத் திட்டம் செய்", "என் வணிக வரைவுத் திட்டம் உருவாக்கு"},
	},
	"funding": {
		"en": {"Find government schemes for me", "Which government schemes can help my business?"},
		"hi": {"मेरे लिए सरकारी योजनाएँ खोजो", "कौन सी सरकारी योजनाएँ मेरे व्यापार में मदद कर सकती हैं?"},
		"te": {"నాకు ప్రభుత్వ పథకాలు వెతకండి", "నా వ్యాపారానికి ఏ ప్రభుత్వ పథకాలు సహాయపడతాయి?"},
		"ta": {"எனக்கான அரசுத் திட்டங்களைத் தேடு", "என் வணிகத்துக்கு எந்த அரசுத் திட்டங்கள் உதவும்?"},
	},
	"apply": {
		"en": {"Draft an application for the best scheme", "Check my eligibility and draft an application for the best-fitting scheme"},
		"hi": {"सबसे अच्छी योजना का आवेदन तैयार करो", "मेरी पात्रता जाँचो और सबसे सही योजना का आवेदन तैयार करो"},
		"te": {"ఉత్తమ పథకానికి దరఖాస్తు తయారు చేయండి", "నా అర్హత చెక్ చేసి సరిపోయే పథకానికి దరఖాస్తు తయారు చేయండి"},
		"ta": {"சிறந்த திட்டத்துக்கு விண்ணப்பம் தயாரி", "என் தகுதியைச் சரிபார்த்து பொருத்தமான திட்டத்துக்கு விண்ணப்பம் தயாரி"},
	},
}

var toolStage = map[string]string{
	"check_scam": "safety", "check_scam_screenshot": "safety", "list_scam_patterns": "safety", "generate_upi_qr": "onboarding",
	"add_records": "records", "get_records_summary": "records", "stock_advice": "records", "build_blueprint": "blueprint", "scheme_intake": "funding",
	"find_schemes": "funding", "search_funding": "funding", "match_funding": "funding", "draft_application": "apply",
}

var nextAfterTool = map[string]string{
	"check_scam": "onboarding", "check_scam_screenshot": "onboarding", "list_scam_patterns": "onboarding",
	"generate_upi_qr": "records", "add_records": "blueprint", "stock_advice": "blueprint", "get_records_summary": "blueprint",
	"build_blueprint": "funding", "search_funding": "apply", "find_schemes": "apply", "match_funding": "apply",
}

func defaultNudge(t *turnState, lang string) *Nudge {
	next := ""
	for i := len(t.steps) - 1; i >= 0 && next == ""; i-- {
		next = nextAfterTool[t.steps[i].Tool]
	}
	if next == "" {
		return nil
	}
	n := defaultNudges[next][lang]
	if n[0] == "" {
		n = defaultNudges[next]["en"]
	}
	return &Nudge{Label: n[0], Prompt: n[1], Stage: next}
}

func slicesContains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

type chatSession struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Stage     string `json:"stage"`
	Summary   string `json:"summary"`
	UpdatedAt string `json:"updated_at"`
}

type chatMessage struct {
	ID        int64          `json:"id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Meta      map[string]any `json:"meta"`
	CreatedAt string         `json:"created_at"`
}

func getSession(ctx context.Context, uid, sid int64) (*chatSession, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	s := &chatSession{}
	err = db.QueryRow(ctx, `SELECT id, title, stage, summary, to_char(updated_at,'YYYY-MM-DD HH24:MI') FROM sessions WHERE id=$1 AND user_id=$2`, sid, uid).
		Scan(&s.ID, &s.Title, &s.Stage, &s.Summary, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpErr(404, "conversation not found")
	}
	return s, err
}

func sessionMessages(ctx context.Context, sid int64, limit int) ([]chatMessage, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT id, role, content, meta, to_char(created_at,'YYYY-MM-DD HH24:MI') FROM
		(SELECT * FROM messages WHERE session_id=$1 ORDER BY id DESC LIMIT $2) m ORDER BY id`, sid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []chatMessage{}
	for rows.Next() {
		var m chatMessage
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.Meta, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

const maxToolRounds = 6

func handleChat(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		SessionID int64  `json:"session_id"`
		Message   string `json:"message"`
		Image     string `json:"image"`
		Source    string `json:"source"` // text | voice | chip
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	in.Message = strings.TrimSpace(in.Message)
	if in.Message == "" && in.Image == "" {
		return httpErr(400, "type or say something")
	}
	if in.Image != "" && !strings.HasPrefix(in.Image, "data:image/") {
		return httpErr(400, "attachment must be an image")
	}
	in.Message = truncate(in.Message, 4000)
	ctx := r.Context()
	db, err := DB()
	if err != nil {
		return err
	}
	var sess *chatSession
	if in.SessionID > 0 {
		if sess, err = getSession(ctx, u.ID, in.SessionID); err != nil {
			return err
		}
	} else {
		title := truncate(in.Message, 60)
		if title == "" {
			title = "Screenshot check"
		}
		sess = &chatSession{Title: title, Stage: "safety"}
		if err := db.QueryRow(ctx, `INSERT INTO sessions(user_id, title) VALUES($1,$2) RETURNING id`, u.ID, title).Scan(&sess.ID); err != nil {
			return err
		}
	}
	history, err := sessionMessages(ctx, sess.ID, 16)
	if err != nil {
		return err
	}
	msgs := []ChatMsg{{Role: "system", Content: systemPrompt(ctx, u, sess)}}
	for _, m := range history {
		c := m.Content
		if steps, ok := m.Meta["steps"].([]any); ok && len(steps) > 0 {
			labels := []string{}
			for _, st := range steps {
				if sm, ok := st.(map[string]any); ok {
					labels = append(labels, fmt.Sprint(sm["label"]))
				}
			}
			c += "\n(tools used: " + strings.Join(labels, "; ") + ")"
		}
		msgs = append(msgs, ChatMsg{Role: m.Role, Content: c})
	}
	userText := in.Message
	if in.Image != "" {
		msgs = append(msgs, ChatMsg{Role: "user", Content: []map[string]any{
			{"type": "text", "text": userText + "\n[User attached an image. If it may be a payment/SMS/QR, call check_scam_screenshot; if it is a bill or khata page, describe entries and call add_records.]"},
			{"type": "image_url", "image_url": map[string]any{"url": in.Image}},
		}})
	} else {
		msgs = append(msgs, ChatMsg{Role: "user", Content: userText})
	}

	t := &turnState{uid: u.ID, lang: u.Lang, image: in.Image, stage: sess.Stage}
	var reply string
	for round := 0; ; round++ {
		tools := agentTools
		if round == maxToolRounds {
			tools = nil // force a final answer
		}
		respMsg, err := chat(ctx, "agent", msgs, tools, nil)
		if err != nil {
			return err
		}
		if len(respMsg.ToolCalls) == 0 {
			reply, _ = respMsg.Content.(string)
			break
		}
		respMsg.Role = "assistant"
		msgs = append(msgs, respMsg)
		for _, c := range respMsg.ToolCalls {
			msgs = append(msgs, ChatMsg{Role: "tool", ToolCallID: c.ID, Content: runTool(ctx, t, c)})
		}
	}

	for i := len(t.steps) - 1; i >= 0; i-- {
		if st, ok := toolStage[t.steps[i].Tool]; ok {
			t.stage = st
			break
		}
	}
	if t.nudge == nil {
		t.nudge = defaultNudge(t, u.Lang)
	}
	storedUser := in.Message
	userMeta := map[string]any{"source": in.Source}
	if in.Image != "" {
		userMeta["image"] = true
	}
	var userMsgID int64
	if err := db.QueryRow(ctx, `INSERT INTO messages(session_id, role, content, meta) VALUES($1,'user',$2,$3) RETURNING id`, sess.ID, storedUser, userMeta).Scan(&userMsgID); err != nil {
		return err
	}
	meta := map[string]any{"steps": t.steps, "cards": t.cards, "nudge": t.nudge, "stage": t.stage}
	am := chatMessage{Role: "assistant", Content: reply, Meta: meta, CreatedAt: time.Now().In(ist).Format("2006-01-02 15:04")}
	if err := db.QueryRow(ctx, `INSERT INTO messages(session_id, role, content, meta) VALUES($1,'assistant',$2,$3) RETURNING id`, sess.ID, reply, meta).Scan(&am.ID); err != nil {
		return err
	}
	sess.Stage = t.stage
	db.Exec(ctx, `UPDATE sessions SET stage=$1, updated_at=now() WHERE id=$2`, t.stage, sess.ID)
	// Rolling summary keeps long sessions coherent across serverless cold starts.
	if len(history)+2 >= 8 && (len(history)+2)%8 == 0 {
		summarizeSession(ctx, sess, append(history, chatMessage{Role: "user", Content: storedUser}, am))
	}
	return writeJSON(w, map[string]any{"session": sess, "user_message_id": userMsgID, "message": am})
}

func summarizeSession(ctx context.Context, sess *chatSession, msgs []chatMessage) {
	var b strings.Builder
	b.WriteString("Previous summary: " + sess.Summary + "\n")
	for _, m := range msgs {
		fmt.Fprintf(&b, "%s: %s\n", m.Role, truncate(m.Content, 500))
	}
	var res struct {
		Summary string `json:"summary"`
	}
	if err := chatJSON(ctx, "summary", "Summarise this merchant-assistant conversation in <=80 English words: concerns raised, actions taken (checks, QR, records, blueprint, matches, drafts), open next steps.",
		b.String(), obj(map[string]any{"summary": str("")}), &res); err != nil {
		return
	}
	if db, err := DB(); err == nil {
		db.Exec(ctx, `UPDATE sessions SET summary=$1 WHERE id=$2`, res.Summary, sess.ID)
	}
}

func handleSessions(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	rows, err := db.Query(r.Context(), `SELECT id, title, stage, summary, to_char(updated_at,'YYYY-MM-DD HH24:MI') FROM sessions WHERE user_id=$1 ORDER BY updated_at DESC LIMIT 30`, u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []chatSession{}
	for rows.Next() {
		var s chatSession
		if err := rows.Scan(&s.ID, &s.Title, &s.Stage, &s.Summary, &s.UpdatedAt); err != nil {
			return err
		}
		out = append(out, s)
	}
	return writeJSON(w, out)
}

func handleSession(w http.ResponseWriter, r *http.Request, u *User) error {
	sid, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	s, err := getSession(r.Context(), u.ID, sid)
	if err != nil {
		return err
	}
	msgs, err := sessionMessages(r.Context(), sid, 100)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"session": s, "messages": msgs})
}

func handleDeleteSession(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	if _, err := db.Exec(r.Context(), `DELETE FROM sessions WHERE id=$1 AND user_id=$2`, r.PathValue("id"), u.ID); err != nil {
		return err
	}
	return writeJSON(w, map[string]bool{"ok": true})
}
