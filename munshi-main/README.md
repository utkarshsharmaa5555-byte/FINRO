# Finro — हिसाब, safe and growing

**Live:** https://munshi-ochre.vercel.app · Synora Track 03 — Financial & Entrepreneurial Empowerment Agent

Finro is a trusted neighbourhood assistant and digital munshi in a merchant's pocket. One conversation takes an
informal micro-entrepreneur from **UPI fraud safety → a real shop QR → a voice-kept khata → a business
blueprint → live government schemes → a filled application draft**, in Telugu, Hindi, Tamil or English.

## Demo accounts

| Persona | Phone | PIN | Story |
|---|---|---|---|
| Lakshmi Devi | `9000000001` | `1111` | Idli cart, Vijayawada · Telugu · profile intentionally incomplete to show the scheme intake |
| Ramesh Gupta | `9000000002` | `2222` | Kirana store, Varanasi · Hindi · wants to launch a namkeen brand |
| Mentor Desk | `9000000000` | `0000` | Admin: source health, scraper runs, AI latency/tokens, tickets |

The login page has one-tap buttons for each.

## 3-minute demo script

1. **Landing** → counters are live from the DB (schemes tracked, scam patterns, last source check). Toggle dark mode and language.
2. Log in as **Lakshmi** (Telugu). Paste: *"Accept this request to receive ₹5000 for your order, enter UPI PIN"*.
   Watch the **tool trail** (`✓ Checked against 12 known scam patterns`), the **SCAM stamp**, steps in Telugu, and the 1930 helpline. Tap 🔊 to hear it.
3. Tap the proactive chip → **UPI QR** is generated → *Print standee* (PDF) or PNG.
4. Say (mic) or type *"ఈరోజు 40 ఇడ్లీలు 800కి అమ్మాను, బియ్యం 300"* → khata entries appear for confirmation → Save.
5. Ask *"నాకు రెండో బండి కొనడానికి ఏ ప్రభుత్వ పథకాలు సహాయపడతాయి?"* → Finro asks only the **missing** details (domain, stage, needs) in a tap card →
   personalised schemes for **Andhra Pradesh** with *why it fits you*, *how to apply*, *confirm on site* and **official links**.
6. Log in as **Ramesh** (Hindi) → **Matching** → *Check my eligibility*: rule-by-rule ✓ ✗ ? with evidence; biotech calls excluded; *Why not* list.
7. **Draft application** → sections pre-filled from the blueprint, financials from the khata, gaps marked `[NEEDED: …]` → DOCX / PDF.
8. **Mentor Desk** → source health, scraper runs, AI calls (p90 latency, tokens), tickets.

## Also in the app

- **Stock & AI buying advice (Khata):** every item carries a run rate (units/day over 7 and 30 days), trend vs the prior three weeks, days of stock left, margin and a computed reorder quantity. The AI decides what to buy more and less of and explains why in the merchant's language; **quantities are computed, never model-generated**, and an item that is out or nearly out can never be listed under "buy less" (its sales only look weak because the shelf is empty).
- **Business overview (`/app/overview`):** today vs 7/30-day averages, profit, udhaar, stock value, what is running out with days left and a suggested order, what sells ranked by revenue with run rates and trends, average sales by weekday with the best day, the 30-day chart, slow movers and the latest AI brief.
- **Blueprint flowchart (`/app/blueprint`):** the plan as a flow — today → opportunity → investment (proportional use-of-funds bar) → 12-month milestones → risks as decision diamonds → vision — with AI-generated illustrations (the chat model writes three scene prompts, then an image model draws them; with no image model on the key, the same model draws them as **sanitised SVG vector art**). The document view remains as a second tab.
- **Khata ↔ stock link:** an entry naming a stock item also moves stock, so run rates and alerts stay live.
- **Scope guardrail:** the agent answers business and money questions only; anything else (health, relationships, sport, coding) gets one polite line and a redirect, while business-adjacent questions stay in scope.

## How it maps to the brief

| Requirement | What Finro does |
|---|---|
| Real-world fraud prevention | 12-pattern catalogue (collect request, fake screenshot, QR-to-receive, KYC SMS, soundbox agent, digital arrest…) with RBI/NPCI/cybercrime sources, **plus a daily Google News feed** classified by the model into patterns. Text and **screenshot (vision)** checks return a verdict, the matched patterns, warning signs found and actions. |
| Actionable onboarding | Real NPCI-format `upi://pay` links (validated VPA, optional amount), PNG QR via `go-qrcode`, printable standee with a scam warning in the merchant's language. |
| Live grant aggregation (2–3 sources) | ① **BIRAC calls for proposals** parsed from HTML (real deadlines) ② **official scheme portals** (PM SVANidhi, MUDRA, Stand-Up India, CGTMSE, AIM) extracted by the LLM **with quote verification** ③ a **scheme directory** of central + state schemes (AP, TN, UP, Telangana) whose official links are re-checked every refresh. Daily Vercel cron, admin refresh, stale-refresh on read; every run logged. |
| Tailored applications, no broad matching | Blueprint → pre-ranking by state/domain/needs → **parallel rule-by-rule eligibility checks** (pass/fail/unknown with evidence) → drafts filled from blueprint + khata, missing facts left as `[NEEDED: …]`. |
| Seamless conversational escalation | One session with a visible stage rail (Safety → QR → Khata → Blueprint → Funding → Apply), tool-driven stage changes, deterministic next-step nudges, rolling session summaries and persistent business memory. |
| Localised financial-literacy UX | Full UI in EN/HI/TE/TA, AI replies in native script, voice input and read-aloud, cached LLM translation for dynamic content. |

### Anti-hallucination design (mentor focus)
- **Quoted, not guessed:** a scraped deadline or amount is stored only if the model's supporting quote appears verbatim on the fetched page (`verifyQuote`, unit-tested). Otherwise the UI says *"Deadline not stated on source"*.
- **UI renders facts from DB rows,** never from model prose; the agent prompt forbids stating deadlines/amounts.
- **Blueprint financials are overwritten server-side** with numbers computed from the khata.
- Eligibility quotes that don't match the scheme text are dropped; model-invented grant IDs are ignored.
- Voice/photo ledger entries are proposals until the user taps Save.

## Architecture

```
React + Vite (ledger design system, 4 languages)  ──►  /api/* on Vercel (Go serverless, Mumbai bom1)
                                                         ├─ agent.go   gpt-4.1-mini tool loop (17 tools), stages, nudges, memory
                                                         ├─ fraud.go   catalogue + news classification + text/vision checks
                                                         ├─ grants.go  BIRAC parser, quote-verified extraction, cron refresh
                                                         ├─ schemes.go scheme directory, intake, personalised finder
                                                         ├─ funding.go blueprint, parallel eligibility scoring, drafts
                                                         ├─ records.go khata, AI entry parsing (text/voice/photo)
                                                         ├─ stock.go    run rates, days-left, reorder qty, AI buying advice, overview
                                                         ├─ visual.go   AI image prompts + generation; svgart.go draws + sanitises SVG
                                                         └─ Neon Postgres (users, sessions, messages, ledger, memories, grants, fits, drafts, runs, ai_calls, translations…)
```

- **Why Go on Vercel:** one small function, fast cold starts, stdlib HTTP routing. Functions run in **Mumbai** because several government portals block non-Indian IPs.
- **IndicTrans2 trade-off:** it can't run inside a serverless function, so translations use the LLM with a Postgres cache (`translations` table).
- **Voice:** browser speech recognition / synthesis (hi-IN, te-IN, ta-IN) first; OpenAI transcription / TTS automatically used when the key has access.

## Run locally

```bash
vercel env pull .env.local        # DATABASE_URL, SESSION_SECRET, CRON_SECRET (+ add OPENAI_API_KEY)
go run ./cmd/dev setup            # migrate + seed
go run ./cmd/dev reset-demo        # restore the three demo personas before judging
go run ./cmd/dev                  # API on :8787
npm --prefix web run dev          # UI on :5173 (proxies /api)
```

## Tests

```bash
go test ./app/                    # UPI links, BIRAC parser fixture, quote verification, grant status,
                                  # stock classification, advice grounding guard, SVG sanitiser
node scripts/prime-demo.mjs       # pre-build each persona's advice, blueprint and illustrations
node scripts/smoke.mjs            # end-to-end against production: auth, khata AI parse, QR, fraud, grants, blueprint, matching, drafts, help, admin
```

## Team study guide

`docs/munshi-qa.pdf` — 19 pages, about 90 questions and answers covering the product, the demo script, how the AI works,
architecture, security, testing, honest limitations and likely judge questions, written so technical and non-technical
teammates can both answer confidently. Regenerate after editing `docs/munshi-qa.html`:

```bash
chrome --headless=new --no-pdf-header-footer --print-to-pdf=docs/munshi-qa.pdf docs/munshi-qa.html
```
