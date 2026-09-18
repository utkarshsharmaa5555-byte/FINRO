package app

import (
	"net/http"
	"sync"
)

var (
	routerOnce sync.Once
	mux        *http.ServeMux
)

func Router() http.Handler {
	routerOnce.Do(func() {
		m := http.NewServeMux()
		m.HandleFunc("GET /api/health", handle(func(w http.ResponseWriter, r *http.Request) error {
			db, err := DB()
			if err == nil {
				err = db.Ping(r.Context())
			}
			return writeJSON(w, map[string]any{"ok": err == nil, "db": err == nil})
		}))
		m.HandleFunc("GET /api/public/stats", handle(handlePublicStats))
		m.HandleFunc("POST /api/setup", handle(handleSetup))
		m.HandleFunc("GET /api/cron/refresh", handle(handleCron))

		m.HandleFunc("POST /api/auth/login", handle(handleLogin))
		m.HandleFunc("POST /api/auth/signup", handle(handleSignup))
		m.HandleFunc("POST /api/auth/logout", handle(handleLogout))
		m.HandleFunc("GET /api/me", authed(handleMe))
		m.HandleFunc("PUT /api/settings", authed(handleSettings))
		m.HandleFunc("DELETE /api/account", authed(handleDeleteAccount))

		m.HandleFunc("POST /api/chat", authed(handleChat))
		m.HandleFunc("GET /api/sessions", authed(handleSessions))
		m.HandleFunc("GET /api/sessions/{id}", authed(handleSession))
		m.HandleFunc("DELETE /api/sessions/{id}", authed(handleDeleteSession))

		m.HandleFunc("GET /api/fraud/patterns", authed(handlePatterns))
		m.HandleFunc("POST /api/fraud/check", authed(handleScamCheck))

		m.HandleFunc("GET /api/qr", authed(handleListQR))
		m.HandleFunc("POST /api/qr", authed(handleCreateQR))
		m.HandleFunc("GET /api/qr/{id}/png", authed(handleQRPNG))

		m.HandleFunc("GET /api/records", authed(handleRecords))
		m.HandleFunc("POST /api/records", authed(handleAddEntries))
		m.HandleFunc("POST /api/records/parse", authed(handleParseEntries))
		m.HandleFunc("DELETE /api/records/{id}", authed(handleDeleteEntry))

		m.HandleFunc("GET /api/overview", authed(handleOverview))
		m.HandleFunc("GET /api/stock", authed(handleStock))
		m.HandleFunc("POST /api/stock", authed(handleCreateItem))
		m.HandleFunc("POST /api/stock/advice", authed(handleStockAdvice))
		m.HandleFunc("PATCH /api/stock/{id}", authed(handleMoveItem))
		m.HandleFunc("DELETE /api/stock/{id}", authed(handleDeleteItem))
		m.HandleFunc("POST /api/blueprint/{id}/illustrate", authed(handleIllustrate))
		m.HandleFunc("GET /api/blueprint/{id}/images", authed(handleBlueprintImages))
		m.HandleFunc("GET /api/blueprint/{id}/image/{slot}", authed(handleBlueprintImage))

		m.HandleFunc("GET /api/profile", authed(handleGetProfile))
		m.HandleFunc("PUT /api/profile", authed(handlePutProfile))
		m.HandleFunc("POST /api/memories", authed(handleAddMemory))
		m.HandleFunc("DELETE /api/memories/{id}", authed(handleDeleteMemory))

		m.HandleFunc("GET /api/grants", authed(handleGrants))
		m.HandleFunc("POST /api/grants/refresh", adminOnly(handleRefresh))
		m.HandleFunc("GET /api/schemes/intake", authed(handleSchemeIntake))
		m.HandleFunc("POST /api/schemes/find", authed(handleFindSchemes))
		m.HandleFunc("GET /api/blueprint", authed(handleGetBlueprint))
		m.HandleFunc("POST /api/blueprint", authed(handleBuildBlueprint))
		m.HandleFunc("GET /api/matches", authed(handleMatches))
		m.HandleFunc("POST /api/matches", authed(handleRunMatches))
		m.HandleFunc("GET /api/drafts", authed(handleListDrafts))
		m.HandleFunc("POST /api/drafts", authed(handleCreateDraft))
		m.HandleFunc("PUT /api/drafts/{id}", authed(handleUpdateDraft))

		m.HandleFunc("GET /api/faqs", authed(handleFAQs))
		m.HandleFunc("POST /api/help/ask", authed(handleHelpAsk))
		m.HandleFunc("GET /api/tickets", authed(handleTickets))
		m.HandleFunc("POST /api/tickets", authed(handleCreateTicket))

		m.HandleFunc("POST /api/voice/transcribe", authed(handleTranscribe))
		m.HandleFunc("POST /api/voice/tts", authed(handleTTS))
		m.HandleFunc("POST /api/translate", authed(handleTranslate))

		m.HandleFunc("GET /api/admin/models", adminOnly(handleModels))
		m.HandleFunc("GET /api/admin/overview", adminOnly(handleAdminOverview))
		mux = m
	})
	return mux
}
