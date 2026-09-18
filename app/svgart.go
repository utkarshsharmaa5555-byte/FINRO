package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// When the API key has no image model (this project only has a chat model), the chat model draws the
// scene as SVG vector art instead. Same prompt, same style, and it is swapped out automatically the
// moment a real image model becomes available.
// ponytail: vector art is simpler than a raster scene; upgrade path is gpt-image-1 in imageModels.

const svgStyle = "Draw in a flat, warm, hand-made ledger style: cream paper background (#F4EDE0), deep indigo ink outlines (#1F2A44), " +
	"vermilion (#C8412B), turmeric (#C8901B), leaf green (#2F6B4F), muted brick (#8E2A1E). Build the scene from layered shapes: " +
	"sky/wall wash, ground line, the main structure (cart, shop front, kiosk, machine), the goods, and 2-4 simple people silhouettes " +
	"(no facial features). Use 40-90 elements, strokeWidth 2-3 for outlines, soft fill opacity for shading."

var svgSchema = obj(map[string]any{
	"svg": str("a complete <svg viewBox=\"0 0 400 300\" xmlns=\"http://www.w3.org/2000/svg\"> … </svg> document"),
})

var (
	svgTagRe    = regexp.MustCompile(`</?\s*([a-zA-Z][a-zA-Z0-9:_-]*)`)
	svgEventRe  = regexp.MustCompile(`(?i)\son\w+\s*=`)
	svgAllowed  = map[string]bool{"svg": true, "g": true, "path": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true, "defs": true, "lineargradient": true, "radialgradient": true, "stop": true, "clippath": true, "use": true, "title": true, "desc": true}
	svgUnsafeRe = regexp.MustCompile(`(?i)(<script|<foreignobject|<image|<iframe|<style|javascript:|data:text/html|xlink:href\s*=\s*"[^#]|href\s*=\s*"[^#])`)
)

// sanitizeSVG returns the document only if it contains nothing but a known-safe subset of SVG.
func sanitizeSVG(s string) (string, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "<svg"); i > 0 {
		s = s[i:] // drop any markdown fence or prose before the document
	}
	if j := strings.LastIndex(s, "</svg>"); j > 0 {
		s = s[:j+len("</svg>")]
	}
	switch {
	case !strings.HasPrefix(s, "<svg"), len(s) > 300_000:
		return "", fmt.Errorf("not an svg document")
	case svgUnsafeRe.MatchString(s), svgEventRe.MatchString(s):
		return "", fmt.Errorf("svg contained unsafe markup")
	}
	for _, m := range svgTagRe.FindAllStringSubmatch(s, -1) {
		if !svgAllowed[strings.ToLower(m[1])] {
			return "", fmt.Errorf("svg contained <%s>", m[1])
		}
	}
	if !strings.Contains(s, "viewBox") {
		s = strings.Replace(s, "<svg", `<svg viewBox="0 0 400 300"`, 1)
	}
	return s, nil
}

func svgIllustration(ctx context.Context, scene string) ([]byte, string, string, error) {
	sys := "You are an illustrator who draws with SVG. Return ONE valid SVG document for the scene described. " +
		svgStyle + " Only these elements: svg, g, path, rect, circle, ellipse, line, polyline, polygon, defs, linearGradient, radialGradient, stop, clipPath, title. " +
		"No text elements, no letters, no embedded images, no scripts, no CSS classes — put presentation in attributes."
	var res struct {
		SVG string `json:"svg"`
	}
	if err := chatJSON(ctx, "svg_art", sys, "Scene: "+scene, svgSchema, &res); err != nil {
		return nil, "", "", err
	}
	clean, err := sanitizeSVG(res.SVG)
	if err != nil {
		return nil, "", "", err
	}
	return []byte(clean), "image/svg+xml", chatModel + " (SVG)", nil
}
