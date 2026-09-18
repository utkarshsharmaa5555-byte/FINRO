package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Scheme directory: well-known government schemes with official links. Summaries are hand-written
// (not model output); amounts/deadlines are deliberately left out — users are sent to the official site.
type dirScheme struct {
	Title, URL, Summary, Eligibility string
	Tags                             []string
}

var schemeDirectory = []dirScheme{
	{"PM SVANidhi — street vendor working-capital loan", "https://pmsvanidhi.mohua.gov.in/",
		"Collateral-free working-capital loans for street vendors in stages, with interest subsidy on timely repayment and cashback for digital transactions.",
		"Street vendors with a vending certificate / ID from the urban local body or a letter of recommendation.", []string{"loan", "street-vendor", "food & street vending", "retail / kirana", "urban", "women", "any-gender"}},
	{"Pradhan Mantri MUDRA Yojana (Shishu / Kishore / Tarun)", "https://www.mudra.org.in/",
		"Collateral-free business loans through banks, NBFCs and MFIs for non-farm micro and small enterprises, in categories by loan size.",
		"Non-farm income-generating micro/small businesses — manufacturing, trading, services and allied agriculture activities.", []string{"loan", "retail / kirana", "food & street vending", "services", "manufacturing", "textiles & tailoring", "beauty & wellness", "transport & logistics", "any-gender"}},
	{"PMEGP — Prime Minister’s Employment Generation Programme", "https://www.kviconline.gov.in/pmegpeportal/pmegphome/index.jsp",
		"Credit-linked subsidy (margin money) for setting up NEW micro-enterprises in manufacturing or services, with higher subsidy for special categories and rural areas.",
		"Individuals above 18; new units only; some education requirement for larger projects; special categories include women, SC/ST, OBC, minorities.", []string{"subsidy", "loan", "idea", "new (under 1 year)", "manufacturing", "services", "food & street vending", "sc-st", "women", "any-gender"}},
	{"Stand-Up India", "https://www.standupmitra.in/",
		"Bank loans for setting up a greenfield enterprise in manufacturing, services, trading or agri-allied activities, with handholding support.",
		"SC/ST and/or women entrepreneurs above 18 starting a new (greenfield) enterprise.", []string{"loan", "women", "sc-st", "idea", "new (under 1 year)"}},
	{"CGTMSE — collateral-free credit guarantee", "https://www.cgtmse.in/",
		"Guarantee cover so banks can lend to micro and small enterprises without collateral or third-party guarantee.",
		"New and existing micro & small enterprises borrowing from member lending institutions.", []string{"loan", "credit-guarantee", "any-gender", "manufacturing", "services", "retail / kirana"}},
	{"PM Vishwakarma", "https://pmvishwakarma.gov.in/",
		"Recognition, skill training with stipend, toolkit incentive, collateral-free credit and marketing support for traditional artisans and craftspeople.",
		"Artisans in 18 listed trades (e.g. carpenter, tailor, potter, cobbler, barber, goldsmith) working with hands and tools, family-based.", []string{"loan", "training", "equipment", "handicraft & artisan", "textiles & tailoring", "beauty & wellness", "any-gender"}},
	{"PMFME — PM Formalisation of Micro Food Processing Enterprises", "https://pmfme.mofpi.gov.in/",
		"Credit-linked capital subsidy, seed capital for SHGs, and branding/marketing support for micro food-processing units (One District One Product focus).",
		"Existing or new micro food-processing units, SHGs, FPOs and cooperatives.", []string{"subsidy", "branding", "market access", "food & street vending", "agriculture & allied", "manufacturing", "women"}},
	{"Udyam Registration (MSME)", "https://udyamregistration.gov.in/",
		"Free online MSME registration using Aadhaar and PAN — the gateway to most MSME schemes, priority-sector loans and government tenders.",
		"Any micro, small or medium enterprise (manufacturing or services).", []string{"registration", "any-gender", "retail / kirana", "food & street vending", "services", "manufacturing", "tech startup"}},
	{"Udyam Assist Platform (informal micro enterprises)", "https://udyamregistration.gov.in/",
		"Helps informal micro enterprises not registered for GST get an Udyam Assist Certificate so they can access priority-sector lending.",
		"Informal micro enterprises exempt from GST, registered through a designated agency.", []string{"registration", "street-vendor", "food & street vending", "retail / kirana"}},
	{"Startup India Seed Fund Scheme (SISFS)", "https://seedfund.startupindia.gov.in/",
		"Seed funding through selected incubators for proof of concept, prototype, trials, market entry and commercialisation.",
		"DPIIT-recognised startups, incorporated recently, with a scalable, technology-backed business idea.", []string{"grant", "tech startup", "idea", "new (under 1 year)", "startup"}},
	{"Credit Guarantee Scheme for Startups (CGSS)", "https://www.ncgtc.in/",
		"Guarantee cover on loans given by banks/NBFCs/venture debt funds to DPIIT-recognised startups.",
		"DPIIT-recognised startups with steady revenue that meet lender criteria.", []string{"loan", "tech startup", "startup", "running (1-5 years)"}},
	{"SIDBI — MSME finance & schemes", "https://www.sidbi.in/",
		"Direct and refinance loans, fund-of-funds and special programmes for micro, small and women-led enterprises.",
		"MSMEs meeting SIDBI programme criteria.", []string{"loan", "women", "manufacturing", "services", "growing (5+ years)", "running (1-5 years)"}},
	{"National SC-ST Hub", "https://www.scsthub.in/",
		"Handholding, capacity building, market linkage, and special credit-linked subsidies for SC/ST entrepreneurs.",
		"Enterprises owned by SC/ST entrepreneurs.", []string{"subsidy", "training", "market access", "sc-st"}},
	{"DAY-NRLM — women’s self-help group credit", "https://aajeevika.gov.in/",
		"Self-help groups of rural women get revolving funds, bank linkage and enterprise support.",
		"Rural women organised into SHGs.", []string{"loan", "women", "rural", "agriculture & allied", "handicraft & artisan", "food & street vending"}},
	{"Agriculture Infrastructure Fund", "https://agriinfra.dac.gov.in/",
		"Interest subvention and credit guarantee for post-harvest and community farming infrastructure projects.",
		"Farmers, FPOs, agri-entrepreneurs, startups, SHGs and cooperatives.", []string{"loan", "subsidy", "agriculture & allied"}},
	{"Agri-Clinics & Agri-Business Centres (ACABC)", "https://www.acabcmis.gov.in/",
		"Free residential training plus credit-linked subsidy to start agri-ventures.",
		"Graduates/diploma holders in agriculture and allied subjects.", []string{"training", "subsidy", "agriculture & allied", "idea"}},
	{"PMKVY — free skill training (Skill India)", "https://www.pmkvyofficial.org/",
		"Free short-term, industry-recognised skill training and certification.",
		"Indian youth; school/college dropouts and unemployed.", []string{"training", "any-gender", "idea"}},
	{"Government e-Marketplace (GeM) — sell to government", "https://gem.gov.in/",
		"Register as a seller to supply products/services directly to government buyers.",
		"Businesses with PAN and Udyam/GST as applicable.", []string{"market access", "manufacturing", "services", "handicraft & artisan", "textiles & tailoring"}},
	{"ONDC — sell online on open network", "https://ondc.org/",
		"Open digital commerce network that lets small shops be discovered on multiple buyer apps.",
		"Sellers onboarded through ONDC seller apps.", []string{"market access", "retail / kirana", "food & street vending"}},
	{"PM Suraksha Bima & Jeevan Jyoti (insurance)", "https://jansuraksha.gov.in/",
		"Very low-premium accident and life insurance through your bank account — protects your family and business.",
		"Bank account holders within the age bands of each scheme.", []string{"insurance", "any-gender", "street-vendor", "retail / kirana", "food & street vending"}},
	{"ZED Certification for MSMEs", "https://zed.msme.gov.in/",
		"Quality certification with subsidy on certification cost; improves credibility with buyers and lenders.",
		"Udyam-registered MSMEs (manufacturing and services).", []string{"subsidy", "manufacturing", "services", "growing (5+ years)", "running (1-5 years)"}},
	{"Tamil Nadu MSME schemes — NEEDS, UYEGP, AABCS", "https://www.msmeonline.tn.gov.in/",
		"State programmes for new entrepreneurs: capital subsidy and loans for educated youth (NEEDS), unemployed youth (UYEGP), and SC/ST entrepreneurs (AABCS).",
		"Residents of Tamil Nadu meeting each scheme's age/education/category rules.", []string{"state:Tamil Nadu", "subsidy", "loan", "idea", "new (under 1 year)", "sc-st"}},
	{"Uttar Pradesh — One District One Product (ODOP) schemes", "https://odopup.in/",
		"Margin-money subsidy, training, toolkits and marketing support for ODOP products of each UP district.",
		"UP residents working in the district's ODOP product sector.", []string{"state:Uttar Pradesh", "subsidy", "training", "market access", "handicraft & artisan", "manufacturing", "food & street vending"}},
	{"Uttar Pradesh MSME & Mukhyamantri Yuva schemes", "https://msme.up.gov.in/",
		"State MSME department schemes including self-employment support for young entrepreneurs.",
		"Residents of Uttar Pradesh meeting scheme criteria.", []string{"state:Uttar Pradesh", "loan", "subsidy", "idea", "new (under 1 year)"}},
	{"Andhra Pradesh Industries — MSME incentives", "https://www.apindustries.gov.in/",
		"State industrial policy incentives for MSMEs, including special support for women and SC/ST entrepreneurs.",
		"Enterprises set up in Andhra Pradesh as per current state policy.", []string{"state:Andhra Pradesh", "subsidy", "manufacturing", "services", "women", "sc-st"}},
	{"Telangana — T-PRIDE & WE Hub (women founders)", "https://wehub.telangana.gov.in/",
		"State support for women-led startups and enterprises: incubation, market access and funding connects.",
		"Women entrepreneurs in Telangana.", []string{"state:Telangana", "women", "tech startup", "training", "market access"}},
}

const directorySource = "Scheme directory"

func seedDirectory(ctx context.Context) error {
	gs := make([]Grant, len(schemeDirectory))
	for i, d := range schemeDirectory {
		gs[i] = Grant{Source: directorySource, SourceURL: d.URL + "#" + slugify(d.Title), Title: d.Title, Summary: d.Summary, Eligibility: d.Eligibility, Tags: d.Tags}
	}
	return upsertGrants(ctx, gs)
}

// checkDirectoryLinks verifies each official link still responds, so the UI can show "link checked".
func checkDirectoryLinks(ctx context.Context) (int, error) {
	db, err := DB()
	if err != nil {
		return 0, err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for _, d := range schemeDirectory {
		wg.Go(func() {
			cctx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			_, err := fetch(cctx, d.URL)
			db.Exec(ctx, `UPDATE grants SET link_ok=$1, link_checked_at=now(), fetched_at=now() WHERE source_url=$2`, err == nil, d.URL+"#"+slugify(d.Title))
			if err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return ok, nil
}

// ---------- intake ----------

var intakeFields = []string{"state", "gender", "age", "social_category", "business_domain", "business_stage", "needs"}

var intakeOptions = map[string][]string{
	"state":           {"Andhra Pradesh", "Telangana", "Tamil Nadu", "Karnataka", "Kerala", "Maharashtra", "Uttar Pradesh", "Bihar", "West Bengal", "Gujarat", "Rajasthan", "Madhya Pradesh", "Delhi", "Odisha", "Punjab", "Haryana", "Assam", "Other"},
	"gender":          {"female", "male", "transgender", "prefer not to say"},
	"social_category": {"General", "OBC", "SC", "ST", "Minority", "prefer not to say"},
	"business_domain": {"food & street vending", "retail / kirana", "agriculture & allied", "manufacturing", "handicraft & artisan", "services", "tech startup", "textiles & tailoring", "beauty & wellness", "transport & logistics", "other"},
	"business_stage":  {"idea", "new (under 1 year)", "running (1-5 years)", "growing (5+ years)"},
	"needs":           {"loan", "subsidy", "grant", "training", "registration", "market access", "insurance", "equipment"},
}

func missingIntake(profile map[string]any) []string {
	var miss []string
	for _, f := range intakeFields {
		v, ok := profile[f]
		switch x := v.(type) {
		case string:
			ok = ok && strings.TrimSpace(x) != ""
		case []any:
			ok = ok && len(x) > 0
		case nil:
			ok = false
		}
		if !ok {
			miss = append(miss, f)
		}
	}
	return miss
}

type SchemeRec struct {
	GrantID     int64    `json:"grant_id"`
	Fit         string   `json:"fit"` // high | medium
	WhyForYou   string   `json:"why_for_you"`
	WhatYouGet  string   `json:"what_you_get"`
	HowToApply  []string `json:"how_to_apply"`
	CheckOnSite []string `json:"check_on_site"`
	Grant       *Grant   `json:"grant,omitempty"`
}

type SchemeResult struct {
	Recommendations []SchemeRec    `json:"recommendations"`
	Suggestions     []string       `json:"suggestions"`
	DocumentsReady  []string       `json:"documents_to_keep_ready"`
	Profile         map[string]any `json:"profile"`
	Considered      int            `json:"considered"`
}

var schemeRecSchema = obj(map[string]any{
	"recommendations": arr(obj(map[string]any{
		"grant_id":      integer(""),
		"fit":           enum("high", "medium"),
		"why_for_you":   str("1-2 sentences tying the scheme to THIS person's state/gender/age/category/domain/stage/needs"),
		"what_you_get":  str("the type of help in plain words; no amounts unless written in the scheme text"),
		"how_to_apply":  arr(str("short practical step")),
		"check_on_site": arr(str("eligibility detail the user must confirm on the official site")),
	})),
	"suggestions":             arr(str("practical next steps e.g. get Udyam registration first, open a current account, keep 6 months of khata")),
	"documents_to_keep_ready": arr(str("")),
})

func stateAllowed(tags []string, state string) bool {
	for _, t := range tags {
		if s, ok := strings.CutPrefix(t, "state:"); ok {
			return strings.EqualFold(s, state)
		}
	}
	return true
}

func findSchemes(ctx context.Context, uid int64, focus, lang string) (*SchemeResult, error) {
	profile, _, err := loadProfile(ctx, uid)
	if err != nil {
		return nil, err
	}
	if miss := missingIntake(profile); len(miss) > 0 {
		return nil, httpErr(422, "missing: "+strings.Join(miss, ", "))
	}
	grants, err := listGrants(ctx)
	if err != nil {
		return nil, err
	}
	state, _ := profile["state"].(string)
	byID := map[int64]Grant{}
	var cands []map[string]any
	for _, g := range grants {
		if g.Status == "closed" || !stateAllowed(g.Tags, state) {
			continue
		}
		byID[g.ID] = g
		cands = append(cands, map[string]any{"grant_id": g.ID, "title": g.Title, "summary": g.Summary, "eligibility": g.Eligibility, "tags": g.Tags, "source": g.Source})
	}
	sys := "You recommend Indian government schemes to a micro-entrepreneur. Choose 3-6 schemes from CANDIDATES only (by grant_id) that genuinely fit the person's " +
		"state, gender, age, social category, business domain, stage and needs. Do not recommend schemes whose eligibility clearly excludes them " +
		"(e.g. SC/ST-only schemes for a General-category person, artisan schemes for a kirana). Prefer one scheme per programme (no duplicates). " +
		"Never state amounts, interest rates or deadlines unless written in the candidate text — send the user to the official site for them. " +
		"Always include Udyam registration in suggestions if they do not list Udyam in documents. Write all text in " + langNames[lang] + " (native script), simple words."
	user := mustJSON(map[string]any{"person": profile, "focus": focus, "candidates": cands})
	var res SchemeResult
	if err := chatJSON(ctx, "find_schemes", sys, user, schemeRecSchema, &res); err != nil {
		return nil, err
	}
	recs := []SchemeRec{}
	seen := map[int64]bool{}
	for _, r := range res.Recommendations {
		g, ok := byID[r.GrantID]
		if !ok || seen[r.GrantID] {
			continue
		}
		seen[r.GrantID] = true
		gc := g
		r.Grant = &gc
		recs = append(recs, r)
	}
	res.Recommendations = recs
	res.Profile = profile
	res.Considered = len(cands)
	return &res, nil
}

func handleSchemeIntake(w http.ResponseWriter, r *http.Request, u *User) error {
	profile, _, err := loadProfile(r.Context(), u.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"missing": missingIntake(profile), "profile": profile, "options": intakeOptions})
}

func handleFindSchemes(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct{ Focus string }
	readJSON(r, &in)
	res, err := findSchemes(r.Context(), u.ID, truncate(in.Focus, 500), u.Lang)
	if err != nil {
		return err
	}
	return writeJSON(w, res)
}

func intakeSummary(profile map[string]any) string {
	var b strings.Builder
	for _, f := range intakeFields {
		fmt.Fprintf(&b, "%s=%v; ", f, profile[f])
	}
	return b.String()
}
