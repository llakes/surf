package browser

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/llakes/surf/errors"
	"github.com/llakes/surf/jar"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

func checkFuncErr(f func() error) {
	err := f()
	if err != nil {
		panic(err)
	}
}

// Attribute represents a Browser capability.
type Attribute int

// AttributeMap represents a map of Attribute values.
type AttributeMap map[Attribute]bool

// File represents a input type file, that includes the fileName and a io.reader
type File struct {
	FileName string
	Data     io.Reader
}

// FileSet represents a map of files used to port multipart
type FileSet map[string]*File

const (
	// SendReferer instructs a Browser to send the Referer header.
	SendReferer Attribute = iota

	// MetaRefreshHandling instructs a Browser to handle the refresh meta tag.
	MetaRefreshHandling

	// FollowRedirects instructs a Browser to follow Location headers.
	FollowRedirects
)

// InitialAssetsSliceSize is the initial size when allocating a slice of page
// assets. Increasing this size may lead to a very small performance increase
// when downloading assets from a page with a lot of assets.
var InitialAssetsSliceSize = 20

// Browsable represents an HTTP web browser.
type Browsable interface {
	// SetUserAgent sets the user agent.
	SetUserAgent(ua string)

	// SetAttribute sets a browser instruction attribute.
	SetAttribute(a Attribute, v bool)

	// SetAttributes is used to set all the browser attributes.
	SetAttributes(a AttributeMap)

	// SetState sets the init browser state.
	SetState(sj *jar.State)

	// State returns the browser state.
	State() *jar.State

	// SetBookmarksJar sets the bookmarks jar the browser uses.
	SetBookmarksJar(bj jar.BookmarksJar)

	// BookmarksJar returns the bookmarks jar the browser uses.
	BookmarksJar() jar.BookmarksJar

	// SetCookieJar is used to set the cookie jar the browser uses.
	SetCookieJar(cj http.CookieJar)

	// CookieJar returns the cookie jar the browser uses.
	CookieJar() http.CookieJar

	// SetHistoryJar is used to set the history jar the browser uses.
	SetHistoryJar(hj jar.History)

	// HistoryJar returns the history jar the browser uses.
	HistoryJar() jar.History

	// SetHeadersJar sets the headers the browser sends with each request.
	SetHeadersJar(h http.Header)

	// SetTimeout sets the timeout for requests.
	SetTimeout(t time.Duration)

	// SetTransport sets the http library transport mechanism for each request.
	SetTransport(rt http.RoundTripper)

	// AddRequestHeader adds a header the browser sends with each request.
	AddRequestHeader(name, value string)

	// Open requests the given URL using the GET method.
	Open(url string) error

	// Open requests the given URL using the HEAD method.
	Head(url string) error

	// OpenForm appends the data values to the given URL and sends a GET request.
	OpenForm(url string, data url.Values) error

	// OpenBookmark calls Get() with the URL for the bookmark with the given name.
	OpenBookmark(name string) error

	// Post requests the given URL using the POST method.
	Post(url string, contentType string, body io.Reader) error

	// PostForm requests the given URL using the POST method with the given data.
	PostForm(url string, data url.Values) error

	// PostMultipart requests the given URL using the POST method with the given data using multipart/form-data format.
	PostMultipart(u string, fields url.Values, files FileSet) error

	// Back loads the previously requested page.
	Back() bool

	// Reload duplicates the last successful request.
	Reload() error

	// Bookmark saves the page URL in the bookmarks with the given name.
	Bookmark(name string) error

	// Click clicks on the page element matched by the given expression.
	Click(expr string) error

	// Form returns the form in the current page that matches the given expr.
	Form(expr string) (Submittable, error)

	// Forms returns an array of every form in the page.
	Forms() []Submittable

	// Links returns an array of every link found in the page.
	Links() []*Link

	// Images returns an array of every image found in the page.
	Images() []*Image

	// Stylesheets returns an array of every stylesheet linked to the document.
	Stylesheets() []*Stylesheet

	// Scripts returns an array of every script linked to the document.
	Scripts() []*Script

	// SiteCookies returns the cookies for the current site.
	SiteCookies() []*http.Cookie

	// ResolveUrl returns an absolute URL for a possibly relative URL.
	ResolveUrl(u *url.URL) *url.URL

	// ResolveStringUrl works just like ResolveUrl, but the argument and return value are strings.
	ResolveStringUrl(u string) (string, error)

	// Url returns the page URL as a string.
	Url() *url.URL

	// StatusCode returns the response status code.
	StatusCode() int

	// ResponseHeaders returns the page headers.
	ResponseHeaders() http.Header

	// Dom returns the inner *html.Node.
	Dom() *html.Node

	// Find returns the dom selections matching the given expression.
	Find(expr string) []*html.Node

	// Find returns the dom selections matching the given expression.
	FindOne(expr string) *html.Node

	// Create a new Browser instance and inherit the configuration
	// Read more: https://github.com/llakes/surf/issues/23
	NewTab() (b *Browser)
}

// Browser is the default Browser implementation.
type Browser struct {
	// HTTP client
	client *http.Client

	// state is the current browser state.
	state *jar.State

	// userAgent is the User-Agent header value sent with requests.
	userAgent string

	// bookmarks stores the saved bookmarks.
	bookmarks jar.BookmarksJar

	// history stores the visited pages.
	history jar.History

	// headers are additional headers to send with each request.
	headers http.Header

	// attributes is the set browser attributes.
	attributes AttributeMap

	// refresh is a timer used to meta refresh pages.
	refresh *time.Timer
}

// buildClient instanciates the *http.Client used by the browser
func (bow *Browser) buildClient() *http.Client {
	cc := &http.Client{
		CheckRedirect: bow.shouldRedirect,
	}

	transport := &http.Transport{}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	cc.Transport = transport
	return cc
}

// Open requests the given URL using the GET method.
func (bow *Browser) Open(u string) error {
	ur, err := url.Parse(u)
	if err != nil {
		return err
	}
	return bow.httpGET(ur, nil)
}

// Head requests the given URL using the HEAD method.
func (bow *Browser) Head(u string) error {
	ur, err := url.Parse(u)
	if err != nil {
		return err
	}
	return bow.httpHEAD(ur, nil)
}

// OpenForm appends the data values to the given URL and sends a GET request.
func (bow *Browser) OpenForm(u string, data url.Values) error {
	ul, err := url.Parse(u)
	if err != nil {
		return err
	}
	ul.RawQuery = data.Encode()

	return bow.Open(ul.String())
}

// OpenBookmark calls Open() with the URL for the bookmark with the given name.
func (bow *Browser) OpenBookmark(name string) error {
	url, err := bow.bookmarks.Read(name)
	if err != nil {
		return err
	}
	return bow.Open(url)
}

// Post requests the given URL using the POST method.
func (bow *Browser) Post(u string, contentType string, body io.Reader) error {
	ur, err := url.Parse(u)
	if err != nil {
		return err
	}
	return bow.httpPOST(ur, bow.Url(), contentType, body)
}

// PostForm requests the given URL using the POST method with the given data.
func (bow *Browser) PostForm(u string, data url.Values) error {
	return bow.Post(u, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// PostMultipart requests the given URL using the POST method with the given data using multipart/form-data format.
func (bow *Browser) PostMultipart(u string, fields url.Values, files FileSet) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, vs := range fields {
		for _, v := range vs {
			writer.WriteField(k, v)
		}
	}
	for k, file := range files {
		fw, err := writer.CreateFormFile(k, file.FileName)
		if err != nil {
			return err
		}
		if file.Data != nil {
			_, err = io.Copy(fw, file.Data)
			if err != nil {
				return err
			}
		}
	}
	err := writer.Close()
	if err != nil {
		return err

	}
	return bow.Post(u, writer.FormDataContentType(), body)
}

// Back loads the previously requested page.
//
// Returns a boolean value indicating whether a previous page existed, and was
// successfully loaded.
func (bow *Browser) Back() bool {
	if bow.history.Len() > 1 {
		bow.state = bow.history.Pop()
		return true
	}
	return false
}

// Reload duplicates the last successful request.
func (bow *Browser) Reload() error {
	if bow.state.Request != nil {
		return bow.httpRequest(bow.state.Request)
	}
	return errors.NewPageNotLoaded("Cannot reload, the previous request failed.")
}

// Bookmark saves the page URL in the bookmarks with the given name.
func (bow *Browser) Bookmark(name string) error {
	return bow.bookmarks.Save(name, bow.ResolveUrl(bow.Url()).String())
}

// Click clicks on the page element matched by the given expression.
//
// Currently this is only useful for click on links, which will cause the browser
// to load the page pointed at by the link. Future versions of Surf may support
// JavaScript and clicking on elements will fire the click event.
func (bow *Browser) Click(expr string) error {
	sel := bow.FindOne(expr)
	if sel == nil {
		return errors.NewElementNotFound(
			"Element not found matching expr '%s'.", expr)
	}

	ur, err := url.Parse(htmlquery.SelectAttr(sel, "href"))
	if err != nil {
		return err
	}
	return bow.httpGET(bow.ResolveUrl(ur), bow.Url())
}

// Form returns the form in the current page that matches the given expr.
func (bow *Browser) Form(expr string) (Submittable, error) {
	sel := bow.FindOne(expr)
	if sel == nil {
		return nil, errors.NewElementNotFound(
			"Form not found matching expr '%s'.", expr)
	}

	return NewForm(bow, sel), nil
}

// Forms returns an array of every form in the page.
func (bow *Browser) Forms() []Submittable {
	sel := bow.Find("//form")
	dlen := len(sel)
	if dlen == 0 {
		return nil
	}

	forms := make([]Submittable, dlen)
	for _, n := range sel {
		forms = append(forms, NewForm(bow, n))
	}
	return forms
}

// Links returns an array of every link found in the page.
func (bow *Browser) Links() []*Link {
	links := make([]*Link, 0, InitialAssetsSliceSize)
	nLinks := bow.Find("//a")
	for _, n := range nLinks {
		href := htmlquery.SelectAttr(n, "href")
		ur, err := url.Parse(href)
		if err == nil {
			links = append(links, NewLinkAsset(
				bow.ResolveUrl(ur),
				htmlquery.SelectAttr(n, "id"),
				htmlquery.InnerText(n),
			))
		}
	}

	return links
}

// Images returns an array of every image found in the page.
func (bow *Browser) Images() []*Image {
	images := make([]*Image, 0, InitialAssetsSliceSize)
	nImages := bow.Find("//img")
	for _, n := range nImages {
		href := htmlquery.SelectAttr(n, "src")
		ur, err := url.Parse(href)
		if err == nil {
			images = append(images, NewImageAsset(
				bow.ResolveUrl(ur),
				htmlquery.SelectAttr(n, "id"),
				htmlquery.SelectAttr(n, "alt"),
				htmlquery.SelectAttr(n, "title"),
			))
		}
	}

	return images
}

// Stylesheets returns an array of every stylesheet linked to the document.
func (bow *Browser) Stylesheets() []*Stylesheet {
	stylesheets := make([]*Stylesheet, 0, InitialAssetsSliceSize)
	nStylesheets := bow.Find("//link")
	for _, n := range nStylesheets {
		rel := htmlquery.SelectAttr(n, "rel")
		if rel == "stylesheet" {
			mAt := htmlquery.SelectAttr(n, "media")
			tAt := htmlquery.SelectAttr(n, "type")
			if mAt == "" {
				mAt = "all"
			}
			if tAt == "" {
				tAt = "text/css"
			}
			href := htmlquery.SelectAttr(n, "href")
			ur, err := url.Parse(href)
			if err == nil {
				stylesheets = append(stylesheets, NewStylesheetAsset(
					bow.ResolveUrl(ur),
					htmlquery.SelectAttr(n, "id"),
					mAt,
					tAt,
				))
			}
		}
	}

	return stylesheets
}

// Scripts returns an array of every script linked to the document.
func (bow *Browser) Scripts() []*Script {
	scripts := make([]*Script, 0, InitialAssetsSliceSize)
	nScripts := bow.Find("//script")
	for _, n := range nScripts {
		tAt := htmlquery.SelectAttr(n, "type")
		if tAt == "" {
			tAt = "text/javascript"
		}
		href := htmlquery.SelectAttr(n, "src")
		ur, err := url.Parse(href)
		if err == nil {
			scripts = append(scripts, NewScriptAsset(
				bow.ResolveUrl(ur),
				htmlquery.SelectAttr(n, "id"),
				tAt,
			))
		}
	}

	return scripts
}

// SiteCookies returns the cookies for the current site.
func (bow *Browser) SiteCookies() []*http.Cookie {
	if bow.client == nil {
		bow.client = bow.buildClient()
	}
	return bow.client.Jar.Cookies(bow.Url())
}

// SetState sets the browser state.
func (bow *Browser) SetState(sj *jar.State) {
	bow.state = sj
}

// State returns the browser state.
func (bow *Browser) State() *jar.State {
	return bow.state
}

// SetCookieJar is used to set the cookie jar the browser uses.
func (bow *Browser) SetCookieJar(cj http.CookieJar) {
	if bow.client == nil {
		bow.client = bow.buildClient()
	}
	bow.client.Jar = cj
}

// CookieJar returns the cookie jar the browser uses.
func (bow *Browser) CookieJar() http.CookieJar {
	if bow.client == nil {
		bow.client = bow.buildClient()
	}
	return bow.client.Jar
}

// SetUserAgent sets the user agent.
func (bow *Browser) SetUserAgent(userAgent string) {
	bow.userAgent = userAgent
}

// SetAttribute sets a browser instruction attribute.
func (bow *Browser) SetAttribute(a Attribute, v bool) {
	bow.attributes[a] = v
}

// SetAttributes is used to set all the browser attributes.
func (bow *Browser) SetAttributes(a AttributeMap) {
	bow.attributes = a
}

// SetBookmarksJar sets the bookmarks jar the browser uses.
func (bow *Browser) SetBookmarksJar(bj jar.BookmarksJar) {
	bow.bookmarks = bj
}

// BookmarksJar returns the bookmarks jar the browser uses.
func (bow *Browser) BookmarksJar() jar.BookmarksJar {
	return bow.bookmarks
}

// SetHistoryJar is used to set the history jar the browser uses.
func (bow *Browser) SetHistoryJar(hj jar.History) {
	bow.history = hj
}

// HistoryJar returns the history jar the browser uses.
func (bow *Browser) HistoryJar() jar.History {
	return bow.history
}

// SetHeadersJar sets the headers the browser sends with each request.
func (bow *Browser) SetHeadersJar(h http.Header) {
	bow.headers = h
}

// SetTransport sets the http library transport mechanism for each request.
// SetTimeout sets the timeout for requests.
func (bow *Browser) SetTimeout(t time.Duration) {
	if bow.client == nil {
		bow.client = bow.buildClient()
	}
	bow.client.Timeout = t
}

// SetTransport sets the http library transport mechanism for each request.
func (bow *Browser) SetTransport(rt http.RoundTripper) {
	if bow.client == nil {
		bow.client = bow.buildClient()
	}
	bow.client.Transport = rt
}

// AddRequestHeader sets a header the browser sends with each request.
func (bow *Browser) AddRequestHeader(name, value string) {
	bow.headers.Set(name, value)
}

// DelRequestHeader deletes a header so the browser will not send it with future requests.
func (bow *Browser) DelRequestHeader(name string) {
	bow.headers.Del(name)
}

// ResolveUrl returns an absolute URL for a possibly relative URL.
func (bow *Browser) ResolveUrl(u *url.URL) *url.URL {
	return bow.Url().ResolveReference(u)
}

// ResolveStringUrl works just like ResolveUrl, but the argument and return value are strings.
func (bow *Browser) ResolveStringUrl(u string) (string, error) {
	pu, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	pu = bow.Url().ResolveReference(pu)
	return pu.String(), nil
}

// Url returns the page URL as a string.
func (bow *Browser) Url() *url.URL {
	if bow.state.Response == nil {
		// there is a possibility that we issued a request, but for
		// whatever reason the request failed.
		if bow.state.Request != nil {
			return bow.state.Request.URL
		}
		return nil
	}

	return bow.state.Response.Request.URL
}

// StatusCode returns the response status code.
func (bow *Browser) StatusCode() int {
	return bow.state.Response.StatusCode
}

// ResponseHeaders returns the page headers.
func (bow *Browser) ResponseHeaders() http.Header {
	return bow.state.Response.Header
}

// OutputHTML returns the page body as a string of html.
func (bow *Browser) OutputHTML() string {
	return htmlquery.OutputHTML(bow.state.Dom, true)
}

// InnerText returns the page body as a string of html.
func (bow *Browser) InnerText() string {
	return htmlquery.InnerText(bow.state.Dom)
}

// Dom returns the inner *html.Node.
func (bow *Browser) Dom() *html.Node {
	return bow.state.Dom
}

// Find returns the dom selections matching the given expression.
func (bow *Browser) Find(expr string) []*html.Node {
	return htmlquery.Find(bow.state.Dom, expr)
}

// FindOne returns the dom selections matching the given expression.
func (bow *Browser) FindOne(expr string) *html.Node {
	return htmlquery.FindOne(bow.state.Dom, expr)
}

// QueryAll returns the dom selections matching the given expression.
func (bow *Browser) QueryAll(expr string) ([]*html.Node, error) {
	return htmlquery.QueryAll(bow.state.Dom, expr)
}

// Query returns the dom selections matching the given expression.
func (bow *Browser) Query(expr string) (*html.Node, error) {
	return htmlquery.Query(bow.state.Dom, expr)
}

func (bow *Browser) NewTab() (b *Browser) {
	b = &Browser{}
	*b = *bow

	return b
}

// buildRequest creates and returns a *http.Request type.
// Sets any headers that need to be sent with the request.
func (bow *Browser) buildRequest(
	method, url string,
	ref *url.URL,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = copyHeaders(bow.headers)

	if host := req.Header.Get("Host"); host != "" {
		req.Host = host
	}
	req.Header.Set("User-Agent", bow.userAgent)
	if bow.attributes[SendReferer] && ref != nil {
		req.Header.Set("Referer", ref.String())
	}
	if os.Getenv("SURF_DEBUG_HEADERS") != "" {
		d, _ := httputil.DumpRequest(req, false)
		fmt.Fprintln(os.Stderr, "===== [DUMP] =====\n", string(d))
	}

	return req, nil
}

func copyHeaders(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	h2 := make(http.Header, len(h))
	for k, v := range h {
		h2[k] = v
	}
	return h2
}

// RawGet makes an HTTP GET request for the given URL.
// When via is not nil, and AttributeSendReferer is true, the Referer header will
// be set to ref.
func (bow *Browser) RawGet(u string) ([]byte, error) {
	var bResp []byte
	req, err := bow.buildRequest("GET", u, nil, nil)
	if err != nil {
		return bResp, err
	}

	if bow.client == nil {
		bow.client = bow.buildClient()
	}
	bow.preSend()

	resp, err := bow.client.Do(req)
	if err != nil {
		return bResp, err
	}
	defer resp.Body.Close()

	var reader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return bResp, err
		}
	case "deflate":
		reader = flate.NewReader(resp.Body)

	default:
		reader = resp.Body
	}

	bResp, err = ioutil.ReadAll(reader)
	if err != nil {
		return bResp, err
	}

	return bResp, nil
}

// httpGET makes an HTTP GET request for the given URL.
// When via is not nil, and AttributeSendReferer is true, the Referer header will
// be set to ref.
func (bow *Browser) httpGET(u *url.URL, ref *url.URL) error {
	req, err := bow.buildRequest("GET", u.String(), ref, nil)
	if err != nil {
		return err
	}
	return bow.httpRequest(req)
}

// httpHEAD makes an HTTP HEAD request for the given URL.
// When via is not nil, and AttributeSendReferer is true, the Referer header will
// be set to ref.
func (bow *Browser) httpHEAD(u *url.URL, ref *url.URL) error {
	req, err := bow.buildRequest("HEAD", u.String(), ref, nil)
	if err != nil {
		return err
	}
	return bow.httpRequest(req)
}

// httpPOST makes an HTTP POST request for the given URL.
// When via is not nil, and AttributeSendReferer is true, the Referer header will
// be set to ref.
func (bow *Browser) httpPOST(u *url.URL, ref *url.URL, contentType string, body io.Reader) error {
	req, err := bow.buildRequest("POST", u.String(), ref, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

	return bow.httpRequest(req)
}

// send uses the given *http.Request to make an HTTP request.
func (bow *Browser) httpRequest(req *http.Request) error {
	var err error
	var resp *http.Response
	var fmSubmit int
	var reader io.Reader
	var dom *html.Node

reqLoop:
	for {
		if bow.client == nil {
			bow.client = bow.buildClient()
		}
		bow.preSend()
		resp, err = bow.client.Do(req)
		if err == nil {
			switch resp.Header.Get("Content-Encoding") {
			case "gzip":
				reader, err = gzip.NewReader(resp.Body)
			case "deflate":
				reader = flate.NewReader(resp.Body)
			default:
				reader = resp.Body
			}

			dom, err = html.Parse(reader)
			if err == nil {
				bow.history.Push(bow.state)
				bow.state = jar.NewHistoryState(req, resp, dom)
				bow.postSend()
			}
		}

		if err == nil || fmSubmit >= 3 {
			break reqLoop
		}
		fmt.Println("request failed with error: ", err)
		fmt.Println("going again...")
		fmSubmit++
		time.Sleep(1 * time.Second)
	}
	if err == nil && resp.Body != nil {
		resp.Body.Close()
	}

	return err
}

// preSend sets browser state before sending a request.
func (bow *Browser) preSend() {
	if bow.refresh != nil {
		bow.refresh.Stop()
	}
}

// postSend sets browser state after sending a request.
func (bow *Browser) postSend() {
	if isContentTypeHtml(bow.state.Response) && bow.attributes[MetaRefreshHandling] {
		n := bow.FindOne("//meta[http-equiv='refresh']")
		if n != nil {
			attr := htmlquery.SelectAttr(n, "content")
			dur, err := time.ParseDuration(attr + "s")
			if err == nil {
				bow.refresh = time.NewTimer(dur)
				go func() {
					<-bow.refresh.C
					bow.Reload()
				}()
			}
		}
	}
}

// shouldRedirect is used as the value to http.Client.CheckRedirect.
func (bow *Browser) shouldRedirect(req *http.Request, _ []*http.Request) error {
	if bow.attributes[FollowRedirects] {
		req.Header.Set("User-Agent", bow.userAgent)
		return nil
	}
	return errors.NewLocation(
		"Redirects are disabled. Cannot follow '%s'.", req.URL.String())
}

// isContentTypeHtml returns true when the given response sent the "text/html" content type.
func isContentTypeHtml(res *http.Response) bool {
	if res != nil {
		ct := res.Header.Get("Content-Type")
		return ct == "" || strings.Contains(ct, "text/html")
	}
	return false
}
