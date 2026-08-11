# API deviations

This file records the deliberate, documented behavioural differences between
`github.com/malcolmston/puppeteer` and the oracle it is measured against in the
cross-language parity harness (`parity/puppeteer/`, oracle `cheerio@1.0.0`).
Each entry is a case where the port and the oracle disagree **and the port is
the more defensible of the two** — so the harness counts them as `deviation`,
not `differ`.

## Selector parsing is stricter than css-what

| parity case | selector | upstream (css-what) | port |
| --- | --- | --- | --- |
| `compile-empty` | `""` | accepts, matches nothing | rejects as malformed |
| `compile-dangling-combinator` | `"div >"` | accepts | rejects (trailing combinator) |
| `compile-bad-attr-op` | `"[a!=b]"` | accepts, treats `!=` as a custom op | rejects (`!=` is not a CSS attribute operator) |

css-what is permissive about malformed input. The port rejects an empty
selector, a selector that ends in a combinator, and the non-standard `!=`
attribute operator. Rejecting invalid CSS is the safer contract for a query
engine, so these are intentional.

## `<select multiple>` uses the option's value attribute, not its text

| parity case |
| --- |
| `form-serialize-select-multiple`, `form-fields-select-multiple`, `form-request-select-multiple`, `form-submit-select-multiple` |

For `<option value="m1">M1</option>`, the port serialises the control's
**value attribute** (`m1`), which is what the HTML form-serialisation algorithm
requires. `cheerio@1.0.0`'s `select.val()` returns the option's **text**
(`M1`) for a `<select multiple>` — a known cheerio bug (also recorded in
`parity/cheerio/parity.json`). The port now emits one name/value pair per
selected option (matching the oracle's *multiplicity*), so the only remaining
difference is the value-vs-text axis, where the port is correct.

## `:empty` follows CSS 2 / Selectors 3, but `Element.IsEmpty` follows Selectors 4

The `:empty` **pseudo-class** matches css-select exactly: an element with
whitespace-only text is *not* `:empty`. The Go-only helper `Element.IsEmpty`
deliberately keeps the Selectors 4 relaxation (whitespace-only counts as empty)
because it reads more naturally as an "is there anything here?" predicate. The
two are intentionally not identical; the parity harness exercises the
pseudo-class, which matches the oracle.

## Serialiser now escapes escapable raw text (security-relevant)

This is a **fix**, not a standing deviation, but it is recorded here because it
changes behaviour relative to the last published release. Through v0.2.0 the
serialiser wrote `<title>`/`<textarea>` content **verbatim**, like true
raw-text `<script>`/`<style>`. Because the parser decodes character references
inside `<title>`/`<textarea>` (they are RCDATA), escaped, inert text was
re-emitted as live markup — a sanitizer-bypass (GHSA-p5fp-ww38-qwfc, harness
findings P8/P9/H2). The serialiser now re-escapes `&`, `<`, `>` (and U+00A0) in
escapable-raw-text elements, so `Page.HTML()`, `Page.NormalizedHTML()`,
`Element.OuterHTML()` and `Element.InnerHTML()` round-trip such content safely.
See `parity/puppeteer/security.json`.
