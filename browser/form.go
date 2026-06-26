package browser

import (
	"net/url"
	"strings"

	"io"

	"github.com/llakes/surf/errors"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// Submittable represents an element that may be submitted, such as a form.
type Submittable interface {
	Method() string
	Action() string
	Input(name, value string) error
	Set(name, value string) error

	// Remove will remove the input completely from the form.
	Remove(name string)

	// RemoveValue will remove a single instance of a form value whose name and value match.
	// This is valuable for removing a single value from a select multiple.
	RemoveValue(name, value string) error

	// Value returns the current value of a form element whose name matches.  If name is not
	// found, error is returned.  For multiple value form element such as select multiple,
	// the first value is returned.
	Value(name string) (string, error)

	// Check will set a checkbox to its active state.  This is done by adding it to
	// the form and setting its value to the value attribute defined in the form.
	Check(name string) error

	// UnCheck will set a checkbox to its inactive state.  This is done by removing
	// it from the form.
	UnCheck(name string) error

	// IsChecked returns a boolean indicating if the checkbox is active or inactive.
	IsChecked(name string) (bool, error)

	// SelectByOptionLabel sets the current value of a select form element acording to the
	// options label.  If the element is a select multiple, multiple options may be selected.
	SelectByOptionLabel(name string, optionLabel ...string) error

	// SelectByOptionValue sets the current value of a select form element acording to the
	// options value.  If the element is a select multiple, multiple options may be selected.
	SelectByOptionValue(name string, optionValue ...string) error

	// SelectValues returns the current values of a form element whose name matches.  If name is not
	// found, error is returned.  For select multiple elements, all values are returned.
	SelectValues(name string) ([]string, error)

	// SelectLabels returns the labels for the selected options for a select form element whose name
	// matches.  If name is not found, error is returned.
	SelectLabels(name string) ([]string, error)

	// File sets the value for an form input type file,
	// it returns an ElementNotFound error if the field does not exists
	File(name string, fileName string, data io.Reader) error

	// SetFile sets the value for a form input type file.
	// It will add the field to the form if necessary
	SetFile(name string, fileName string, data io.Reader)

	Click(button string) error
	ClickByValue(name, value string) error
	Submit() error
	Dom() *html.Node
	Text() string
}

// Form is the default form element.
type Form struct {
	bow       Browsable
	selection *html.Node
	method    string
	action    string
	fields    url.Values
	buttons   url.Values
	checkboxs url.Values
	selects   selects
	files     FileSet
}

// NewForm creates and returns a *Form type.
func NewForm(bow Browsable, s *html.Node) *Form {
	fields, buttons, checkboxs, selects, files := serializeForm(s)
	method, action := formAttributes(bow, s)

	return &Form{
		bow:       bow,
		selection: s,
		method:    method,
		action:    action,
		fields:    fields,
		buttons:   buttons,
		checkboxs: checkboxs,
		selects:   selects,
		files:     files,
	}
}

// Method returns the form method, eg "GET" or "POST".
func (f *Form) Method() string {
	return f.method
}

// Action returns the form action URL.
// The URL will always be absolute.
func (f *Form) Action() string {
	return f.action
}

// Input sets the value of a form field.
// it returns an ElementNotFound error if the field does not exist
func (f *Form) Input(name, value string) error {
	if _, ok := f.fields[name]; ok {
		f.fields.Set(name, value)
		return nil
	}
	return errors.NewElementNotFound("No input found with name '%s'.", name)
}

// File sets the value for an form input type file,
// it returns an ElementNotFound error if the field does not exists
func (f *Form) File(name string, fileName string, data io.Reader) error {

	if _, ok := f.files[name]; ok {
		f.files[name] = &File{FileName: fileName, Data: data}
		return nil
	}
	return errors.NewElementNotFound(
		"No input type 'file' found with name '%s'.", name)
}

// SetFile sets the value for a form input type file.
// It will add the field to the form if necessary
func (f *Form) SetFile(name string, fileName string, data io.Reader) {
	f.files[name] = &File{FileName: fileName, Data: data}
}

// Set will set the value of a form field if it exists,
// or create and set it if it does not.
func (f *Form) Set(name, value string) error {
	if _, ok := f.fields[name]; !ok {
		f.fields.Add(name, value)
		return nil
	}
	return f.Input(name, value)
}

// Check sets the checkbox value to its active state.
func (f *Form) Check(name string) error {
	if _, ok := f.checkboxs[name]; ok {
		f.fields.Set(name, f.checkboxs.Get(name))
		return nil
	}
	return errors.NewElementNotFound("No checkbox found with name '%s'.", name)
}

// UnCheck sets the checkbox value to inactive state.
func (f *Form) UnCheck(name string) error {
	if _, ok := f.checkboxs[name]; ok {
		f.fields.Del(name)
		return nil
	}
	return errors.NewElementNotFound("No checkbox found with name '%s'.", name)
}

// IsChecked returns the current state of the checkbox
func (f *Form) IsChecked(name string) (bool, error) {
	if _, ok := f.checkboxs[name]; ok {
		_, found := f.fields[name]
		return found, nil
	}
	return false, errors.NewElementNotFound("No checkbox found with name '%s'.", name)
}

// Remove will remove the form field if it exists.
func (f *Form) Remove(name string) {
	f.fields.Del(name)
}

// Value returns the current value of a form element whose name matches.  If name is not
// found, error is returned.  For multiple value form element such as select multiple,
// the first value is returned.
func (f *Form) Value(name string) (string, error) {
	if _, ok := f.fields[name]; ok {
		return f.fields.Get(name), nil
	}
	return "", errors.NewElementNotFound("No input found with name '%s'.", name)
}

// RemoveValue will remove a single instance of a form value whose name and value match.
// This is valuable for removing a single value from a select multiple.
func (f *Form) RemoveValue(name, val string) error {
	if _, ok := f.fields[name]; !ok {
		return errors.NewElementNotFound("No input found with name '%s'.", name)
	}
	var save []string
	for _, v := range f.fields[name] {
		if v != val {
			save = append(save, v)
		}
	}
	if len(save) == 0 {
		f.fields.Del(name)
	} else {
		f.fields[name] = save
	}
	return nil
}

// SelectByOptionLabel sets the current value of a select form element acording to the
// options label.  If the element is a select multiple, multiple options may be selected.
func (f *Form) SelectByOptionLabel(name string, optionLabel ...string) error {
	s, ok := f.selects[name]
	if !ok {
		return errors.NewElementNotFound("No select element found with name '%s'.", name)
	}
	if len(optionLabel) > 1 && !s.multiple {
		return errors.NewElementNotFound(
			"The select element with name '%s' is not a select miltiple.",
			name,
		)
	}
	f.fields.Del(name)

	for _, l := range optionLabel {
		if _, ok := s.labels[l]; !ok {
			return errors.NewElementNotFound(
				"The select element with name %q does not have an option with label %q",
				name,
				l,
			)
		}
		f.fields.Add(name, s.labels.Get(l))
	}
	return nil
}

// SelectByOptionValue sets the current value of a select form element acording to the
// options value.  If the element is a select multiple, multiple options may be selected.
func (f *Form) SelectByOptionValue(name string, optionValue ...string) error {
	s, ok := f.selects[name]
	if !ok {
		return errors.NewElementNotFound("No select element found with name '%s'.", name)
	}
	if len(optionValue) > 1 && !s.multiple {
		return errors.NewElementNotFound(
			"The select element with name '%s' is not a select miltiple.",
			name,
		)
	}
	f.fields.Del(name)
	for _, v := range optionValue {
		if _, ok := s.values[v]; !ok {
			return errors.NewElementNotFound(
				"The select element with name %q does not have an option with value %q",
				name,
				v,
			)
		}
		f.fields.Add(name, v)
	}
	return nil
}

// SelectValues returns the current values of a form element whose name matches.  If name is not
// found, error is returned.  For select multiple elements, all values are returned.
func (f *Form) SelectValues(name string) ([]string, error) {
	if _, ok := f.fields[name]; ok {
		return f.fields[name], nil
	}
	return nil, errors.NewElementNotFound("No input found with name '%s'.", name)
}

// SelectLabels returns the labels for the selected options for a select form element whose name
// matches.  If name is not found, error is returned.
func (f *Form) SelectLabels(name string) ([]string, error) {
	s, ok := f.selects[name]
	if !ok {
		return nil, errors.NewElementNotFound("No select element found with name '%s'.", name)
	}
	var labels []string
	for _, v := range f.fields[name] {
		labels = append(labels, s.values.Get(v))
	}
	return labels, nil
}

// Submit submits the form.
// Clicks the first button in the form, or submits the form without using
// any button when the form does not contain any buttons.
func (f *Form) Submit() error {
	if len(f.buttons) > 0 {
		for name := range f.buttons {
			return f.Click(name)
		}
	}
	return f.send("", "")
}

// Click submits the form by clicking the button with the given name.
func (f *Form) Click(button string) error {
	if _, ok := f.buttons[button]; !ok {
		return errors.NewInvalidFormValue(
			"Form does not contain a button with the name '%s'.", button)
	}
	return f.send(button, f.buttons[button][0])
}

// Click submits the form by clicking the button with the given name and value.
func (f *Form) ClickByValue(name, value string) error {
	if _, ok := f.buttons[name]; !ok {
		return errors.NewInvalidFormValue(
			"Form does not contain a button with the name '%s'.", name)
	}
	valueNotFound := true
	for _, val := range f.buttons[name] {
		if val == value {
			valueNotFound = false
			break
		}
	}
	if valueNotFound {
		return errors.NewInvalidFormValue(
			"Form does not contain a button with the name '%s' and value '%s'.", name, value)
	}
	return f.send(name, value)
}

// Dom returns the inner *html.Node.
func (f *Form) Dom() *html.Node {
	return f.selection
}

// Dom returns the inner *html.Node.
func (f *Form) Text() string {
	return htmlquery.OutputHTML(f.selection, true)
}

// send submits the form.
func (f *Form) send(buttonName, buttonValue string) error {
	method := htmlquery.SelectAttr(f.selection, "method")
	if method == "" {
		method = "GET"
	}
	action := htmlquery.SelectAttr(f.selection, "action")
	if action == "" {
		action = f.bow.Url().String()
	}
	aurl, err := url.Parse(action)
	if err != nil {
		return err
	}
	aurl = f.bow.ResolveUrl(aurl)

	values := make(url.Values, len(f.fields)+1)
	for name, vals := range f.fields {
		values[name] = vals
	}
	if buttonName != "" {
		values.Set(buttonName, buttonValue)
	}

	if strings.ToUpper(method) == "GET" {
		return f.bow.OpenForm(aurl.String(), values)
	}
	enctype := htmlquery.SelectAttr(f.selection, "enctype")
	if enctype == "multipart/form-data" {
		return f.bow.PostMultipart(aurl.String(), values, f.files)
	}
	return f.bow.PostForm(aurl.String(), values)
}

// serializeForm converts the form fields into a url.Values type.
// Returns two url.Value types. The first is the form field values, and the
// second is the form button values.
func serializeForm(sel *html.Node) (url.Values, url.Values, url.Values, selects, FileSet) {
	fields := make(url.Values)
	buttons := make(url.Values)
	checkboxs := make(url.Values)
	selects := make(selects)
	files := make(FileSet)
	sels := htmlquery.Find(sel, "//input|//button|//textarea")
	for _, s := range sels {
		if strings.ToLower(htmlquery.SelectAttr(s, "disabled")) != "disabled" {
			name := htmlquery.SelectAttr(s, "name")
			if name != "" {
				val := htmlquery.SelectAttr(s, "value")
				t := htmlquery.SelectAttr(s, "type")
				t = strings.ToLower(t)
				if t == "submit" {
					buttons.Add(name, val)
				} else if t == "checkbox" || t == "radio" {
					if htmlquery.SelectAttr(s, "checked") == "checked" {
						fields.Add(name, val)
					}
					if t == "checkbox" {
						checkboxs.Add(name, val)
					}
				} else if t == "file" {
					files[name] = &File{}
				} else {
					fields.Add(name, val)
				}
			}
		}
	}

	nsels := htmlquery.Find(sel, "//select")
	for _, s := range nsels {
		if strings.ToLower(htmlquery.SelectAttr(s, "disabled")) != "disabled" {
			name := htmlquery.SelectAttr(s, "name")
			if name != "" {
				multiple := strings.Contains(htmlquery.OutputHTML(s, true), "multiple=")
				selects[name] = selectOptions{
					multiple: multiple,
					values:   make(url.Values),
					labels:   make(url.Values),
				}
				var foundSelected bool
				nss := htmlquery.Find(sel, "//option")
				for _, ss := range nss {
					val := htmlquery.SelectAttr(ss, "value")
					l := htmlquery.InnerText(ss)
					selects[name].values.Add(val, strings.TrimSpace(html.UnescapeString(l)))
					selects[name].labels.Add(strings.TrimSpace(html.UnescapeString(l)), val)
					if strings.ToLower(htmlquery.SelectAttr(ss, "selected")) == "selected" {
						if multiple || (!multiple && !foundSelected) {
							fields.Add(name, val)
							foundSelected = true
						}
					}
				}
			}
		}
	}

	return fields, buttons, checkboxs, selects, files
}

type selects map[string]selectOptions

type selectOptions struct {
	multiple bool
	values   url.Values
	labels   url.Values
}

func formAttributes(bow Browsable, s *html.Node) (string, string) {
	method := htmlquery.SelectAttr(s, "method")
	if method == "" {
		method = "GET"
	}
	action := htmlquery.SelectAttr(s, "action")
	if action == "" {
		action = bow.Url().String()
	}
	aurl, err := url.Parse(action)
	if err != nil {
		return "", ""
	}
	aurl = bow.ResolveUrl(aurl)

	return strings.ToUpper(method), aurl.String()
}
