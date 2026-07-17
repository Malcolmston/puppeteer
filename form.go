package puppeteer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Form is a discovered <form> together with its resolved action, method and the
// current values of its fields. It can build (and optionally send) the request
// that submitting it would produce.
type Form struct {
	page    *Page
	element *Element

	// Action is the absolute submission URL.
	Action string
	// Method is the upper-cased HTTP method ("GET" or "POST").
	Method string
	// EncType is the form's encoding; only application/x-www-form-urlencoded is
	// produced when building requests.
	EncType string

	fields []*formField
}

type formField struct {
	name     string
	value    string
	disabled bool
	// included controls whether the field contributes to the submission. For
	// checkboxes and radios this reflects the "checked" attribute.
	included bool
}

// Forms discovers every <form> on the page.
func (p *Page) Forms() ([]*Form, error) {
	els, err := p.QuerySelectorAll("form")
	if err != nil {
		return nil, err
	}
	forms := make([]*Form, 0, len(els))
	for _, el := range els {
		forms = append(forms, p.buildForm(el))
	}
	return forms, nil
}

// FormBySelector returns the first form matching selector, or an error if none
// match.
func (p *Page) FormBySelector(selector string) (*Form, error) {
	el, err := p.QuerySelector(selector)
	if err != nil {
		return nil, err
	}
	if el == nil {
		return nil, fmt.Errorf("puppeteer: no form matches %q", selector)
	}
	if el.TagName() != "form" {
		inner, err := el.QuerySelector("form")
		if err != nil {
			return nil, err
		}
		if inner == nil {
			return nil, fmt.Errorf("puppeteer: element %q is not a form", selector)
		}
		el = inner
	}
	return p.buildForm(el), nil
}

// buildForm extracts action, method and controls from a form element.
func (p *Page) buildForm(el *Element) *Form {
	action := el.AttrOr("action", "")
	actionURL := action
	if resolved, err := p.resolve(action); err == nil {
		actionURL = resolved.String()
	}
	method := strings.ToUpper(strings.TrimSpace(el.AttrOr("method", "GET")))
	if method != http.MethodPost {
		method = http.MethodGet
	}
	f := &Form{
		page:    p,
		element: el,
		Action:  actionURL,
		Method:  method,
		EncType: el.AttrOr("enctype", "application/x-www-form-urlencoded"),
	}
	f.collectFields()
	return f
}

// collectFields walks the form's controls and records their default state.
func (f *Form) collectFields() {
	inputs, _ := f.element.QuerySelectorAll("input")
	for _, in := range inputs {
		name, ok := in.Attr("name")
		if !ok || name == "" {
			continue
		}
		_, disabled := in.Attr("disabled")
		typ := strings.ToLower(in.AttrOr("type", "text"))
		field := &formField{name: name, disabled: disabled}
		switch typ {
		case "checkbox", "radio":
			_, checked := in.Attr("checked")
			field.included = checked && !disabled
			field.value = in.AttrOr("value", "on")
		case "submit", "button", "reset", "image", "file":
			// Buttons are not auto-submitted; skip by default.
			field.included = false
			field.value = in.AttrOr("value", "")
		default:
			field.included = !disabled
			field.value = in.AttrOr("value", "")
		}
		f.fields = append(f.fields, field)
	}

	textareas, _ := f.element.QuerySelectorAll("textarea")
	for _, ta := range textareas {
		name, ok := ta.Attr("name")
		if !ok || name == "" {
			continue
		}
		_, disabled := ta.Attr("disabled")
		f.fields = append(f.fields, &formField{
			name:     name,
			value:    ta.TextContent(),
			disabled: disabled,
			included: !disabled,
		})
	}

	selects, _ := f.element.QuerySelectorAll("select")
	for _, sel := range selects {
		name, ok := sel.Attr("name")
		if !ok || name == "" {
			continue
		}
		_, disabled := sel.Attr("disabled")
		f.fields = append(f.fields, &formField{
			name:     name,
			value:    selectedOptionValue(sel),
			disabled: disabled,
			included: !disabled,
		})
	}
}

// selectedOptionValue returns the value of a <select>'s selected option, or the
// first option's value when none is explicitly selected.
func selectedOptionValue(sel *Element) string {
	options, _ := sel.QuerySelectorAll("option")
	var first string
	for i, opt := range options {
		val := opt.AttrOr("value", opt.TextContent())
		if i == 0 {
			first = val
		}
		if _, ok := opt.Attr("selected"); ok {
			return val
		}
	}
	return first
}

// FieldNames returns the names of all controls in document order.
func (f *Form) FieldNames() []string {
	names := make([]string, 0, len(f.fields))
	for _, fld := range f.fields {
		names = append(names, fld.name)
	}
	return names
}

// Get returns the current value of the named field and whether it exists.
func (f *Form) Get(name string) (string, bool) {
	for _, fld := range f.fields {
		if fld.name == name {
			return fld.value, true
		}
	}
	return "", false
}

// Set assigns a value to the named field, marking it for inclusion. A new field
// is created if the name is unknown, which allows adding parameters the static
// markup did not declare.
func (f *Form) Set(name, value string) {
	for _, fld := range f.fields {
		if fld.name == name {
			fld.value = value
			fld.included = true
			fld.disabled = false
			return
		}
	}
	f.fields = append(f.fields, &formField{name: name, value: value, included: true})
}

// Values returns the url.Values that this form would submit.
func (f *Form) Values() url.Values {
	vals := url.Values{}
	for _, fld := range f.fields {
		if fld.disabled || !fld.included {
			continue
		}
		vals.Add(fld.name, fld.value)
	}
	return vals
}

// FillForm looks up a form by selector, applies values and returns it ready to
// build or submit. It mirrors Puppeteer's ergonomics of filling a form in one
// call.
func (p *Page) FillForm(selector string, values map[string]string) (*Form, error) {
	f, err := p.FormBySelector(selector)
	if err != nil {
		return nil, err
	}
	for k, v := range values {
		f.Set(k, v)
	}
	return f, nil
}

// BuildRequest constructs the *http.Request that submitting the form produces.
// For GET the values are encoded into the query string; for POST they form a
// urlencoded body with the appropriate Content-Type. The request is not sent.
func (f *Form) BuildRequest(ctx context.Context) (*http.Request, error) {
	if f.Action == "" {
		return nil, fmt.Errorf("puppeteer: form has no action URL")
	}
	vals := f.Values()
	if f.Method == http.MethodGet {
		u, err := url.Parse(f.Action)
		if err != nil {
			return nil, fmt.Errorf("puppeteer: parsing action %q: %w", f.Action, err)
		}
		q := u.Query()
		for k, vs := range vals {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("puppeteer: building request: %w", err)
		}
		return req, nil
	}
	body := vals.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.Action, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("puppeteer: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// Submit builds the form request, sends it through the browser and loads the
// response into the originating page.
func (f *Form) Submit() (*Response, error) {
	return f.SubmitContext(context.Background())
}

// SubmitContext is Submit with an explicit context.
func (f *Form) SubmitContext(ctx context.Context) (*Response, error) {
	req, err := f.BuildRequest(ctx)
	if err != nil {
		return nil, err
	}
	return f.page.do(req)
}
