// Package surf ensembles other packages into a usable browser.
package surf

import (
	"fmt"
	"github.com/llakes/surf/agent"
	"github.com/llakes/surf/browser"
	"github.com/llakes/surf/jar"
)

var (
	// DefaultSendReferer is the global value for the AttributeSendReferer attribute.
	DefaultSendReferer = true

	// DefaultMetaRefreshHandling is the global value for the AttributeHandleRefresh attribute.
	DefaultMetaRefreshHandling = true

	// DefaultFollowRedirects is the global value for the AttributeFollowRedirects attribute.
	DefaultFollowRedirects = true

	// DefaultMaxHistoryLength is the global value for max history length.
	DefaultMaxHistoryLength = 0
)

// NewBrowser creates and returns a *browser.Browser type.
func NewBrowser() *browser.Browser {
	bow := &browser.Browser{}
	userAgent := agent.Create()
	fmt.Println("using user agent: ", userAgent)
	bow.SetUserAgent(userAgent)
	bow.SetState(&jar.State{})
	bow.SetCookieJar(jar.NewMemoryCookies())
	bow.SetBookmarksJar(jar.NewMemoryBookmarks())
	hist := jar.NewMemoryHistory()
	hist.SetMax(DefaultMaxHistoryLength)
	bow.SetHistoryJar(hist)
	bow.SetHeadersJar(jar.NewMemoryHeaders())
	bow.SetAttributes(browser.AttributeMap{
		browser.SendReferer:         DefaultSendReferer,
		browser.MetaRefreshHandling: DefaultMetaRefreshHandling,
		browser.FollowRedirects:     DefaultFollowRedirects,
	})

	return bow
}
