package puppeteer

import (
	"strconv"
	"strings"
)

// tokenKind enumerates the token types produced by the tokenizer.
type tokenKind int

const (
	tokText tokenKind = iota
	tokStartTag
	tokEndTag
	tokComment
	tokDoctype
)

// token is a single lexical unit of an HTML byte stream.
type token struct {
	kind        tokenKind
	data        string      // tag name, text, comment body, or doctype name
	attr        []Attribute // start tags only
	selfClosing bool        // start tags only
}

// tokenizer scans an HTML string into a sequence of tokens. It is intentionally
// pragmatic: it handles tags, attributes (quoted, single-quoted and unquoted),
// comments, doctype declarations, raw-text elements (script/style) and
// character references, which covers the overwhelming majority of real markup.
type tokenizer struct {
	src string
	pos int
}

func newTokenizer(src string) *tokenizer { return &tokenizer{src: src} }

// next returns the next token and false when the input is exhausted.
func (t *tokenizer) next() (token, bool) {
	if t.pos >= len(t.src) {
		return token{}, false
	}
	if t.src[t.pos] == '<' {
		if tok, ok := t.readTag(); ok {
			return tok, true
		}
	}
	return t.readText(), true
}

// readText consumes character data up to the next '<' that begins a tag.
func (t *tokenizer) readText() token {
	start := t.pos
	for t.pos < len(t.src) {
		if t.src[t.pos] == '<' && t.looksLikeTag() {
			break
		}
		t.pos++
	}
	raw := t.src[start:t.pos]
	return token{kind: tokText, data: decodeEntities(raw)}
}

// looksLikeTag reports whether the '<' at the current position starts markup.
func (t *tokenizer) looksLikeTag() bool {
	if t.pos+1 >= len(t.src) {
		return false
	}
	c := t.src[t.pos+1]
	return c == '/' || c == '!' || c == '?' || isASCIILetter(c)
}

// readTag parses a tag, comment or doctype beginning at the current '<'. It
// returns ok=false only when the '<' does not actually introduce markup, in
// which case the caller falls back to text.
func (t *tokenizer) readTag() (token, bool) {
	if !t.looksLikeTag() {
		return token{}, false
	}
	// Comment or doctype.
	if t.src[t.pos+1] == '!' {
		if strings.HasPrefix(t.src[t.pos:], "<!--") {
			return t.readComment(), true
		}
		if len(t.src) >= t.pos+2 && (t.src[t.pos+2] == 'd' || t.src[t.pos+2] == 'D') {
			return t.readDoctype(), true
		}
		// Bogus comment (e.g. <![CDATA[ ... ]] or unknown declaration).
		return t.readBogusComment(), true
	}
	if t.src[t.pos+1] == '?' {
		return t.readBogusComment(), true
	}

	end := t.src[t.pos+1] == '/'
	i := t.pos + 1
	if end {
		i++
	}
	nameStart := i
	for i < len(t.src) && isNameChar(t.src[i]) {
		i++
	}
	name := strings.ToLower(t.src[nameStart:i])
	if name == "" {
		return token{}, false
	}
	t.pos = i

	tok := token{data: name}
	if end {
		tok.kind = tokEndTag
	} else {
		tok.kind = tokStartTag
	}

	// Parse attributes until '>' (or self-closing '/>').
	for t.pos < len(t.src) {
		t.skipSpace()
		if t.pos >= len(t.src) {
			break
		}
		c := t.src[t.pos]
		if c == '>' {
			t.pos++
			break
		}
		if c == '/' {
			tok.selfClosing = true
			t.pos++
			continue
		}
		a, ok := t.readAttribute()
		if !ok {
			t.pos++
			continue
		}
		if tok.kind == tokStartTag {
			tok.attr = append(tok.attr, a)
		}
	}
	return tok, true
}

// readAttribute parses a single name[=value] attribute.
func (t *tokenizer) readAttribute() (Attribute, bool) {
	start := t.pos
	for t.pos < len(t.src) && isAttrNameChar(t.src[t.pos]) {
		t.pos++
	}
	if t.pos == start {
		return Attribute{}, false
	}
	name := strings.ToLower(t.src[start:t.pos])
	t.skipSpace()
	if t.pos >= len(t.src) || t.src[t.pos] != '=' {
		return Attribute{Name: name, Value: ""}, true
	}
	t.pos++ // consume '='
	t.skipSpace()
	if t.pos >= len(t.src) {
		return Attribute{Name: name, Value: ""}, true
	}
	var val string
	switch t.src[t.pos] {
	case '"', '\'':
		quote := t.src[t.pos]
		t.pos++
		vs := t.pos
		for t.pos < len(t.src) && t.src[t.pos] != quote {
			t.pos++
		}
		val = t.src[vs:t.pos]
		if t.pos < len(t.src) {
			t.pos++ // consume closing quote
		}
	default:
		vs := t.pos
		for t.pos < len(t.src) && !isSpace(t.src[t.pos]) && t.src[t.pos] != '>' {
			t.pos++
		}
		val = t.src[vs:t.pos]
	}
	return Attribute{Name: name, Value: decodeEntities(val)}, true
}

func (t *tokenizer) readComment() token {
	t.pos += len("<!--")
	end := strings.Index(t.src[t.pos:], "-->")
	var body string
	if end < 0 {
		body = t.src[t.pos:]
		t.pos = len(t.src)
	} else {
		body = t.src[t.pos : t.pos+end]
		t.pos += end + len("-->")
	}
	return token{kind: tokComment, data: body}
}

func (t *tokenizer) readBogusComment() token {
	t.pos++ // consume '<'
	end := strings.IndexByte(t.src[t.pos:], '>')
	var body string
	if end < 0 {
		body = t.src[t.pos:]
		t.pos = len(t.src)
	} else {
		body = t.src[t.pos : t.pos+end]
		t.pos += end + 1
	}
	return token{kind: tokComment, data: body}
}

func (t *tokenizer) readDoctype() token {
	end := strings.IndexByte(t.src[t.pos:], '>')
	var body string
	if end < 0 {
		body = t.src[t.pos:]
		t.pos = len(t.src)
	} else {
		body = t.src[t.pos : t.pos+end]
		t.pos += end + 1
	}
	body = strings.TrimPrefix(body, "<")
	body = strings.TrimSpace(body[len("!"):])
	if len(body) >= len("doctype") {
		body = strings.TrimSpace(body[len("doctype"):])
	}
	return token{kind: tokDoctype, data: body}
}

// readRawText consumes literal text until the matching end tag of a raw-text
// element (script/style). It returns the text and leaves the position at the
// '<' of the closing tag.
func (t *tokenizer) readRawText(tag string) string {
	closing := "</" + tag
	start := t.pos
	if idx := strings.Index(strings.ToLower(t.src[t.pos:]), closing); idx < 0 {
		t.pos = len(t.src)
	} else {
		t.pos += idx
	}
	return t.src[start:t.pos]
}

func (t *tokenizer) skipSpace() {
	for t.pos < len(t.src) && isSpace(t.src[t.pos]) {
		t.pos++
	}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameChar(c byte) bool {
	return isASCIILetter(c) || (c >= '0' && c <= '9') || c == '-' || c == ':' || c == '_'
}

func isAttrNameChar(c byte) bool {
	return !isSpace(c) && c != '=' && c != '>' && c != '/' && c != '<' && c != '"' && c != '\''
}

// decodeEntities replaces HTML character references with their runes.
func decodeEntities(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '&' {
			sb.WriteByte(s[i])
			i++
			continue
		}
		// Find the terminating ';' within a reasonable window.
		semi := -1
		for j := i + 1; j < len(s) && j < i+32; j++ {
			if s[j] == ';' {
				semi = j
				break
			}
			if !isNameChar(s[j]) && s[j] != '#' {
				break
			}
		}
		if semi < 0 {
			sb.WriteByte('&')
			i++
			continue
		}
		ref := s[i+1 : semi]
		if decoded, ok := decodeRef(ref); ok {
			sb.WriteString(decoded)
			i = semi + 1
			continue
		}
		sb.WriteByte('&')
		i++
	}
	return sb.String()
}

func decodeRef(ref string) (string, bool) {
	if ref == "" {
		return "", false
	}
	if ref[0] == '#' {
		var n int64
		var err error
		if len(ref) > 1 && (ref[1] == 'x' || ref[1] == 'X') {
			n, err = strconv.ParseInt(ref[2:], 16, 32)
		} else {
			n, err = strconv.ParseInt(ref[1:], 10, 32)
		}
		if err != nil || n < 0 {
			return "", false
		}
		return string(rune(n)), true
	}
	if v, ok := namedEntities[ref]; ok {
		return v, true
	}
	return "", false
}

// namedEntities is a compact table of the most common named references.
var namedEntities = map[string]string{
	"amp":    "&",
	"lt":     "<",
	"gt":     ">",
	"quot":   "\"",
	"apos":   "'",
	"nbsp":   " ",
	"copy":   "©",
	"reg":    "®",
	"trade":  "™",
	"mdash":  "—",
	"ndash":  "–",
	"hellip": "…",
	"lsquo":  "‘",
	"rsquo":  "’",
	"ldquo":  "“",
	"rdquo":  "”",
	"laquo":  "«",
	"raquo":  "»",
	"middot": "·",
	"bull":   "•",
	"deg":    "°",
	"euro":   "€",
	"pound":  "£",
	"cent":   "¢",
	"yen":    "¥",
	"sect":   "§",
	"para":   "¶",
	"times":  "×",
	"divide": "÷",
}
