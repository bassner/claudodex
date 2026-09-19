package codex

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// chatGPTRoutingCookieStore is process-wide because HTTP requests and WebSocket
// reconnects are made by short-lived Client values. Only infrastructure cookie
// names are admitted; account, session, and authentication cookies are never
// retained here.
type chatGPTRoutingCookieStore struct {
	jar http.CookieJar
}

var sharedChatGPTRoutingCookies = newChatGPTRoutingCookieStore()

func newChatGPTRoutingCookieStore() *chatGPTRoutingCookieStore {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &chatGPTRoutingCookieStore{jar: jar}
}

func (s *chatGPTRoutingCookieStore) addRequestCookies(rawURL string, header http.Header) {
	if s == nil || header == nil || len(header.Values("Cookie")) > 0 {
		return
	}
	requestURL, ok := chatGPTRoutingCookieURL(rawURL)
	if !ok {
		return
	}
	cookies := make([]string, 0)
	for _, cookie := range s.jar.Cookies(requestURL) {
		if isAllowedChatGPTInfrastructureCookie(cookie.Name) {
			cookies = append(cookies, cookie.String())
		}
	}
	if len(cookies) > 0 {
		header.Set("Cookie", strings.Join(cookies, "; "))
	}
}

func (s *chatGPTRoutingCookieStore) storeResponseCookies(rawURL string, header http.Header) {
	if s == nil || header == nil {
		return
	}
	responseURL, ok := chatGPTRoutingCookieURL(rawURL)
	if !ok {
		return
	}
	response := http.Response{Header: header}
	filtered := make([]*http.Cookie, 0)
	for _, cookie := range response.Cookies() {
		if isAllowedChatGPTInfrastructureCookie(cookie.Name) {
			filtered = append(filtered, cookie)
		}
	}
	if len(filtered) > 0 {
		s.jar.SetCookies(responseURL, filtered)
	}
}

func chatGPTRoutingCookieURL(rawURL string) (*url.URL, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, false
	}
	if strings.EqualFold(parsed.Scheme, "wss") {
		parsed.Scheme = "https"
	}
	if !strings.EqualFold(parsed.Scheme, "https") || !isAllowedChatGPTHost(parsed.Hostname()) {
		return nil, false
	}
	return parsed, true
}

func isAllowedChatGPTHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	return host == "chatgpt.com" || strings.HasSuffix(host, ".chatgpt.com") ||
		host == "chat.openai.com" || strings.HasSuffix(host, ".chat.openai.com")
}

func isAllowedChatGPTInfrastructureCookie(name string) bool {
	switch name {
	case "__cf_bm", "__cflb", "__cfruid", "__cfseq", "__cfwaitingroom", "__oailb", "_cfuvid", "cf_clearance", "cf_ob_info", "cf_use_ob":
		return true
	default:
		return strings.HasPrefix(name, "cf_chl_")
	}
}

type routingCookieRoundTripper struct {
	base http.RoundTripper
}

func (t routingCookieRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	request := req.Clone(req.Context())
	request.Header = req.Header.Clone()
	sharedChatGPTRoutingCookies.addRequestCookies(request.URL.String(), request.Header)
	resp, err := t.base.RoundTrip(request)
	if resp != nil {
		sharedChatGPTRoutingCookies.storeResponseCookies(request.URL.String(), resp.Header)
	}
	return resp, err
}

func (c Client) routingCookieHTTPClient() *http.Client {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	clone := *client
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = routingCookieRoundTripper{base: transport}
	return &clone
}
