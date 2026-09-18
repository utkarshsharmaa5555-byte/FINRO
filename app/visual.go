package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Three illustrated moments of the blueprint flowchart.
var imageSlots = []string{"today", "investment", "vision"}

// One visual language for every image so the flowchart reads as one hand-made ledger book.
const imageStyle = "Style: warm hand-drawn ink and watercolour illustration on cream ledger paper, deep indigo ink lines, vermilion and turmeric accents, " +
	"gentle Indian street-market atmosphere, soft morning light, friendly and dignified, square composition. " +
	"Absolutely no text, letters, numbers, logos or signage writing anywhere in the image."

var promptSchema = obj(map[string]any{
	"today":      str("scene of the business as it is today: the place, the goods, the customers (40-70 words)"),
	"investment": str("scene of the planned investment being put to use, e.g. a new cart, machine, shelves or packing table (40-70 words)"),
	"vision":     str("scene of the business 12 months from now after the plan succeeds (40-70 words)"),
})

// imagePrompts asks the chat model to turn a blueprint into three concrete, culturally specific scenes.
func imagePrompts(ctx context.Context, uid int64, b *Blueprint) (map[string]string, error) {
	profile, _, _ := loadProfile(ctx, uid)
	sys := "You write prompts for an image model that illustrates an Indian micro-business plan. Describe concrete visual scenes (objects, place, people's activity, time of day) " +
		"true to the city and trade. People are shown respectfully, from a distance or side, no close-up faces. Never ask for text in the image."
	var out map[string]string
	user := mustJSON(map[string]any{"profile": map[string]any{"business_type": profile["business_type"], "city": profile["city"], "state": profile["state"]}, "blueprint": b.Data})
	if err := chatJSON(ctx, "image_prompts", sys, user, promptSchema, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type imageModel struct {
	name string
	body func(prompt string) map[string]any
	mime string
}

var imageModels = []imageModel{
	{"gpt-image-1-mini", func(p string) map[string]any {
		return map[string]any{"model": "gpt-image-1-mini", "prompt": p, "size": "1024x1024", "quality": "low", "output_format": "webp", "output_compression": 72}
	}, "image/webp"},
	{"gpt-image-1", func(p string) map[string]any {
		return map[string]any{"model": "gpt-image-1", "prompt": p, "size": "1024x1024", "quality": "low", "output_format": "webp", "output_compression": 72}
	}, "image/webp"},
	{"dall-e-3", func(p string) map[string]any {
		return map[string]any{"model": "dall-e-3", "prompt": p, "size": "1024x1024", "quality": "standard"}
	}, "image/png"},
	{"dall-e-2", func(p string) map[string]any {
		return map[string]any{"model": "dall-e-2", "prompt": truncate(p, 950), "size": "512x512"}
	}, "image/png"},
}

var (
	workingModelMu sync.Mutex
	workingModel   = -1 // index into imageModels that last succeeded; -1 unknown
)

// generateImage tries image models in order (keys differ in access) and remembers the first that works.
func generateImage(ctx context.Context, prompt string) ([]byte, string, string, error) {
	workingModelMu.Lock()
	startAt := max(workingModel, 0)
	workingModelMu.Unlock()
	var lastErr error
	for i := startAt; i < len(imageModels); i++ {
		m := imageModels[i]
		start := time.Now()
		b, _ := json.Marshal(m.body(prompt))
		resp, err := openaiDo(ctx, "/images/generations", "application/json", bytes.NewReader(b))
		if err != nil {
			logAI("image", start, 0, false)
			return nil, "", "", err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
		resp.Body.Close()
		var out struct {
			Data []struct {
				B64 string `json:"b64_json"`
				URL string `json:"url"`
			} `json:"data"`
		}
		if resp.StatusCode == 200 && json.Unmarshal(body, &out) == nil && len(out.Data) > 0 {
			var img []byte
			var err error
			if out.Data[0].B64 != "" {
				img, err = base64.StdEncoding.DecodeString(out.Data[0].B64)
			} else {
				img, err = fetch(ctx, out.Data[0].URL) // some models return a short-lived URL
			}
			if err == nil && len(img) > 0 {
				logAI("image", start, 0, true)
				workingModelMu.Lock()
				workingModel = i
				workingModelMu.Unlock()
				return img, m.mime, m.name, nil
			}
		}
		logAI("image", start, 0, false)
		lastErr = fmt.Errorf("%s: %d %s", m.name, resp.StatusCode, truncate(string(body), 200))
		log.Print("image generation: ", lastErr)
	}
	// No raster image model on this key: let the chat model draw the scene as vector art.
	img, mime, model, svgErr := svgIllustration(ctx, prompt)
	if svgErr == nil {
		return img, mime, model, nil
	}
	if lastErr == nil {
		lastErr = svgErr
	}
	return nil, "", "", fmt.Errorf("%w (svg fallback: %v)", lastErr, svgErr)
}

type SlotResult struct {
	Slot   string `json:"slot"`
	OK     bool   `json:"ok"`
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Error  string `json:"error,omitempty"`
}

// illustrateBlueprint generates any missing slot images in parallel and stores them with their prompts.
func illustrateBlueprint(ctx context.Context, uid, blueprintID int64, force bool) ([]SlotResult, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	b := &Blueprint{}
	if err := db.QueryRow(ctx, `SELECT id, data FROM blueprints WHERE id=$1 AND user_id=$2`, blueprintID, uid).Scan(&b.ID, &b.Data); err != nil {
		return nil, httpErr(404, "blueprint not found")
	}
	have := map[string]bool{}
	if !force {
		rows, err := db.Query(ctx, `SELECT slot FROM blueprint_images WHERE blueprint_id=$1`, blueprintID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var s string
			if rows.Scan(&s) == nil {
				have[s] = true
			}
		}
		rows.Close()
	}
	results := make([]SlotResult, len(imageSlots))
	var todo []int
	for i, s := range imageSlots {
		results[i] = SlotResult{Slot: s, OK: have[s]}
		if !have[s] {
			todo = append(todo, i)
		}
	}
	if len(todo) == 0 {
		return results, nil
	}
	prompts, err := imagePrompts(ctx, uid, b)
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	for _, i := range todo {
		wg.Go(func() {
			slot := imageSlots[i]
			prompt := prompts[slot] + "\n" + imageStyle
			img, mime, model, err := generateImage(ctx, prompt)
			results[i].Prompt = prompts[slot]
			if err != nil {
				results[i].Error = truncate(err.Error(), 180)
				return
			}
			if _, err := db.Exec(ctx, `INSERT INTO blueprint_images(blueprint_id, slot, mime, img, prompt, model) VALUES($1,$2,$3,$4,$5,$6)
				ON CONFLICT (blueprint_id, slot) DO UPDATE SET mime=EXCLUDED.mime, img=EXCLUDED.img, prompt=EXCLUDED.prompt, model=EXCLUDED.model, created_at=now()`,
				blueprintID, slot, mime, img, prompts[slot], model); err != nil {
				results[i].Error = "could not save image"
				return
			}
			results[i].OK, results[i].Model = true, model
		})
	}
	wg.Wait()
	return results, nil
}

func handleIllustrate(w http.ResponseWriter, r *http.Request, u *User) error {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	res, err := illustrateBlueprint(r.Context(), u.ID, id, r.URL.Query().Get("force") == "1")
	if err != nil {
		return err
	}
	return writeJSON(w, map[string]any{"slots": res})
}

func handleBlueprintImages(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	rows, err := db.Query(r.Context(), `SELECT bi.slot, bi.model, bi.prompt, extract(epoch from bi.created_at)::bigint FROM blueprint_images bi
		JOIN blueprints b ON b.id = bi.blueprint_id WHERE bi.blueprint_id=$1 AND b.user_id=$2`, r.PathValue("id"), u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var slot, model, prompt string
		var ts int64
		if err := rows.Scan(&slot, &model, &prompt, &ts); err != nil {
			return err
		}
		out = append(out, map[string]any{"slot": slot, "model": model, "prompt": prompt, "v": ts})
	}
	return writeJSON(w, out)
}

func handleBlueprintImage(w http.ResponseWriter, r *http.Request, u *User) error {
	db, err := DB()
	if err != nil {
		return err
	}
	var mime string
	var img []byte
	if err := db.QueryRow(r.Context(), `SELECT bi.mime, bi.img FROM blueprint_images bi JOIN blueprints b ON b.id = bi.blueprint_id
		WHERE bi.blueprint_id=$1 AND bi.slot=$2 AND b.user_id=$3`, r.PathValue("id"), r.PathValue("slot"), u.ID).Scan(&mime, &img); err != nil {
		return httpErr(404, "image not found")
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, err = w.Write(img)
	return err
}

// handleModels lists the models this API key can use — a quick way to see which image/voice models exist.
func handleModels(w http.ResponseWriter, r *http.Request, u *User) error {
	req, err := http.NewRequestWithContext(r.Context(), "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return err
	}
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		key = os.Getenv("OPEN_API_KEY")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := aiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err := json.Unmarshal(body, &out); err != nil {
		return writeJSON(w, map[string]any{"status": resp.StatusCode, "raw": truncate(string(body), 500)})
	}
	ids := make([]string, len(out.Data))
	for i, m := range out.Data {
		ids[i] = m.ID
	}
	return writeJSON(w, map[string]any{"status": resp.StatusCode, "models": ids})
}
