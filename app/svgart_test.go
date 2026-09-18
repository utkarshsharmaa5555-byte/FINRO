package app

import "testing"

func TestSanitizeSVG(t *testing.T) {
	good := `Here you go: <svg viewBox="0 0 400 300"><g><rect x="0" y="0" width="400" height="300" fill="#F4EDE0"/><path d="M10 10h20" stroke="#1F2A44"/></g></svg> done`
	out, err := sanitizeSVG(good)
	if err != nil || !hasPrefix(out, "<svg") || !hasSuffix(out, "</svg>") {
		t.Fatalf("clean svg rejected: %v %q", err, out)
	}
	bad := []string{
		`<svg><script>alert(1)</script></svg>`,
		`<svg><rect onload="alert(1)"/></svg>`,
		`<svg><image href="http://x/y.png"/></svg>`,
		`<svg><foreignObject><b>hi</b></foreignObject></svg>`,
		`<svg><a xlink:href="http://x">go</a></svg>`,
		`<svg><text>hello</text></svg>`,
		`not an svg at all`,
	}
	for _, b := range bad {
		if _, err := sanitizeSVG(b); err == nil {
			t.Errorf("accepted unsafe svg: %s", b)
		}
	}
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
func hasSuffix(s, p string) bool { return len(s) >= len(p) && s[len(s)-len(p):] == p }
