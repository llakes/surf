// Package agent generates user agents strings for well known browsers
// and for custom browsers.
//
// When submitting patches to add user agents formats, please *always* include
// "{{.Coms}}" between the opening ( and closing ) braces, even if you're
// sure the browser would never have additional comments.
package agent

import (
	"bytes"
	"math/rand"
	"runtime"
	"strings"
	"text/template"
)

var (
	bStrings = []string{"Chrome", "Firefox"}
	oStrings = []int{Windows, Linux}
	vStrings = map[string][]string{
		"chrome":  {"141", "140", "139", "138", "137", "136", "135", "134"},
		"firefox": {"144", "143", "142", "141", "140", "139", "138", "137", "136", "135", "134"},
	}

	wVersions = []string{"11.0", "10.0"}
	lVersions = []string{"6.18", "6.17", "6.16", "6.15", "6.14", "6.13", "6.12", "6.11"}
	warch     = []string{"x64", "x86_64"}
	larch     = []string{"x64", "x86_64", "arm64", "aarch64"}

	// Comments are additional comments to add to a user agent string.
	Comments = []string{runtime.Version()}
)

const (
	// Windows operating system.
	Windows int = iota
	// Linux based operating system.
	Linux
)

// TemplateData structure for template data.
type TemplateData struct {
	Name string
	Ver  string
	OSN  string
	OSV  string
	Coms string
}

// OSAttributes stores OS attributes.
type OSAttributes struct {
	OSName    string
	OSVersion string
	Comments  []string
}

// Formats is a collection of UA format strings.
type Formats map[string]string

// UATable is a collection of UAData values.
type UATable map[string]Formats

// Database is the "database" of user agents.
var Database = UATable{
	"chrome": Formats{
		"141": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
		"140": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
		"139": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
		"138": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
		"137": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
		"136": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
		"135": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
		"134": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36",
	},
	"firefox": Formats{
		"144": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:31.0) Gecko/20100101 Firefox/{{.Ver}}",
		"143": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"142": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"141": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"140": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"139": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"138": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"137": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"136": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"135": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
		"134": "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}; rv:30.0) Gecko/20120101 Firefox/{{.Ver}}",
	},
}

func getBrowserVersion(bn string) string {
	bv := vStrings[bn]
	return bv[rand.Intn(len(bv))]
}

func getOSAttributes(os int) OSAttributes {

	var oa OSAttributes
	switch os {
	case Windows:
		oa = OSAttributes{"Windows NT", wVersions[rand.Intn(len(wVersions))], []string{warch[rand.Intn(len(warch))]}}
	case Linux:
		oa = OSAttributes{"Linux", lVersions[rand.Intn(len(lVersions))], []string{larch[rand.Intn(len(larch))]}}
	default:
		oa = OSAttributes{"Windows NT", "11.0", []string{"x64"}}
	}

	return oa
}

// createFromDefaults returns a user agent string using default values.
func createFromDefaults(browser string) string {
	bn := strings.ToLower(browser)
	bv := getBrowserVersion(bn)
	os := oStrings[rand.Intn(len(oStrings))]
	osAttribs := getOSAttributes(os)

	return createFromDetails(
		browser,
		bv,
		osAttribs.OSName,
		osAttribs.OSVersion,
		osAttribs.Comments)
}

// Create generates and returns a complete user agent string.
func Create() string {
	bs := bStrings[rand.Intn(len(bStrings))]
	return createFromDefaults(bs)
}

// Format returns the format string for the given browser name and version.
//
// When a format can't be found for a version, the first format string for the browser
// is returned. When a format can't be found for the browser the default format is
// returned.
func Format(bname, bver string) string {
	bname = strings.ToLower(bname)
	majVer := strings.Split(bver, ".")[0]
	data, ok := Database[bname]
	if ok {
		format, ok := data[majVer]
		if ok {
			return format
		}
	}

	return "Mozilla/5.0 ({{.OSN}} {{.OSV}}{{.Coms}}) Chrome/{{.Ver}} Safari/537.36"
}

// createFromDetails generates and returns a complete user agent string.
func createFromDetails(bname, bver, osname, osver string, c []string) string {
	comments := strings.Join(c, "; ")
	if comments != "" {
		comments = "; " + comments
	}

	data := TemplateData{bname, bver, osname, osver, comments}
	buff := &bytes.Buffer{}
	t := template.New("formatter")
	t.Parse(Format(bname, bver))
	t.Execute(buff, data)

	return buff.String()
}
