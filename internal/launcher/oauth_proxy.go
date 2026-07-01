package launcher

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type OAuthProxy struct {
	listener  net.Listener
	serverTLS tls.Certificate
	caPath    string
	target    string
	logPath   string
	closed    chan struct{}
	once      sync.Once
}

const oauthProxyCertificateLifetime = 397 * 24 * time.Hour

func StartOAuthProxy(target string) (*OAuthProxy, error) {
	cert, caPEM, err := generateOAuthProxyCertificate()
	if err != nil {
		return nil, err
	}
	caFile, err := os.CreateTemp("", "claudodex-ca-*.pem")
	if err != nil {
		return nil, err
	}
	caPath := caFile.Name()
	if _, err := caFile.Write(caPEM); err != nil {
		_ = caFile.Close()
		_ = os.Remove(caPath)
		return nil, err
	}
	if err := caFile.Chmod(0o600); err != nil {
		_ = caFile.Close()
		_ = os.Remove(caPath)
		return nil, err
	}
	if err := caFile.Close(); err != nil {
		_ = os.Remove(caPath)
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = os.Remove(caPath)
		return nil, err
	}
	proxy := &OAuthProxy{
		listener:  listener,
		serverTLS: cert,
		caPath:    caPath,
		target:    strings.TrimRight(target, "/"),
		logPath:   os.Getenv("CLAUDODEX_OAUTH_PROXY_LOG"),
		closed:    make(chan struct{}),
	}
	go proxy.serve()
	return proxy, nil
}

func (p *OAuthProxy) ProxyURL() string {
	if p == nil || p.listener == nil {
		return ""
	}
	return "http://" + p.listener.Addr().String()
}

func (p *OAuthProxy) CAPath() string {
	if p == nil {
		return ""
	}
	return p.caPath
}

func (p *OAuthProxy) Close() error {
	var err error
	p.once.Do(func() {
		close(p.closed)
		if p.listener != nil {
			err = p.listener.Close()
		}
		if p.caPath != "" {
			_ = os.Remove(p.caPath)
		}
	})
	return err
}

func (p *OAuthProxy) serve() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.closed:
				return
			default:
				continue
			}
		}
		go p.handle(conn)
	}
}

func (p *OAuthProxy) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Minute))
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		p.log("read CONNECT failed: " + err.Error())
		return
	}
	if req.Method != http.MethodConnect {
		p.log("non-CONNECT " + req.Method + " " + req.Host)
		_ = writeProxyStatus(conn, http.StatusMethodNotAllowed)
		return
	}
	host := canonicalConnectHost(req.Host)
	p.log("CONNECT " + host)
	if host != "api.anthropic.com:443" {
		p.tunnel(conn, reader, host)
		return
	}
	if err := writeProxyStatus(conn, http.StatusOK); err != nil {
		return
	}
	tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{p.serverTLS}, MinVersion: tls.VersionTLS12})
	defer tlsConn.Close()
	if err := tlsConn.Handshake(); err != nil {
		p.log("TLS handshake failed: " + err.Error())
		return
	}
	p.handleAnthropicTLS(tlsConn)
}

func (p *OAuthProxy) tunnel(client net.Conn, reader *bufio.Reader, host string) {
	upstream, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		p.log("tunnel dial failed " + host + ": " + err.Error())
		_ = writeProxyStatus(client, http.StatusBadGateway)
		return
	}
	defer upstream.Close()
	if err := writeProxyStatus(client, http.StatusOK); err != nil {
		return
	}
	if reader.Buffered() > 0 {
		buffered, _ := reader.Peek(reader.Buffered())
		_, _ = upstream.Write(buffered)
		_, _ = reader.Discard(len(buffered))
	}
	done := make(chan struct{}, 2)
	go proxyCopy(done, upstream, client)
	go proxyCopy(done, client, upstream)
	<-done
}

func proxyCopy(done chan<- struct{}, dst net.Conn, src net.Conn) {
	_, _ = io.Copy(dst, src)
	_ = dst.SetDeadline(time.Now())
	done <- struct{}{}
}

func (p *OAuthProxy) handleAnthropicTLS(conn *tls.Conn) {
	reader := bufio.NewReader(conn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			p.log("read Anthropic request failed: " + err.Error())
			return
		}
		p.log(req.Method + " " + req.URL.RequestURI())
		upgraded := isUpgradeRequest(req)
		if err := p.writeAnthropicResponse(conn, req); err != nil {
			p.log("write Anthropic response failed: " + err.Error())
			return
		}
		if upgraded {
			return
		}
		if req.Close {
			return
		}
	}
}

func (p *OAuthProxy) log(line string) {
	if p == nil || p.logPath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p.logPath), 0o700)
	f, err := os.OpenFile(p.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339Nano), line)
}

func (p *OAuthProxy) writeAnthropicResponse(w io.Writer, req *http.Request) error {
	path := req.URL.Path
	switch oauthProxyRouteFor(req.Method, path) {
	case oauthProxyRouteLocal:
		return p.forwardToLocalProxy(w, req)
	case oauthProxyRouteAnthropic:
		if isUpgradeRequest(req) {
			conn, ok := w.(net.Conn)
			if !ok {
				return writeHTTPResponse(w, http.StatusBadGateway, "application/json", []byte(`{"error":{"type":"api_error","message":"remote-control upgrade unavailable"}}`))
			}
			return p.forwardAnthropicUpgrade(conn, req)
		}
		return p.forwardToAnthropicAPI(w, req)
	default:
		return writeHTTPResponse(w, http.StatusNotFound, "application/json", []byte(`{"error":{"type":"not_found_error","message":"route not provided by Claudodex"}}`))
	}
}

func (p *OAuthProxy) forwardToLocalProxy(w io.Writer, in *http.Request) error {
	targetURL, err := url.Parse(p.target + in.URL.RequestURI())
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(context.Background(), in.Method, targetURL.String(), in.Body)
	if err != nil {
		return err
	}
	copyForwardHeaders(req.Header, in.Header)
	resp, err := oauthProxyForwardClient.Do(req)
	if err != nil {
		return writeHTTPResponse(w, http.StatusBadGateway, "application/json", []byte(`{"error":{"type":"api_error","message":"local Claudodex proxy unavailable"}}`))
	}
	defer resp.Body.Close()
	p.logForwardedResponse("local proxy", in, resp)
	return writeForwardedResponse(w, resp)
}

func (p *OAuthProxy) forwardToAnthropicAPI(w io.Writer, in *http.Request) error {
	targetURL, err := url.Parse(firstPartyAnthropicBaseURL + in.URL.RequestURI())
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(context.Background(), in.Method, targetURL.String(), in.Body)
	if err != nil {
		return err
	}
	copyForwardHeaders(req.Header, in.Header)
	req.Host = "api.anthropic.com"
	resp, err := oauthProxyForwardClient.Do(req)
	if err != nil {
		p.log("direct Anthropic forward failed " + in.Method + " " + in.URL.RequestURI() + ": " + err.Error())
		return writeHTTPResponse(w, http.StatusBadGateway, "application/json", []byte(`{"error":{"type":"api_error","message":"Anthropic Remote Control API unavailable"}}`))
	}
	defer resp.Body.Close()
	p.logForwardedResponse("direct Anthropic", in, resp)
	return writeForwardedResponse(w, resp)
}

func (p *OAuthProxy) logForwardedResponse(target string, req *http.Request, resp *http.Response) {
	if p == nil || resp == nil {
		return
	}
	p.log(fmt.Sprintf("%s %s %s -> %d", target, req.Method, req.URL.RequestURI(), resp.StatusCode))
}

func (p *OAuthProxy) forwardAnthropicUpgrade(client net.Conn, in *http.Request) error {
	upstream, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", "api.anthropic.com:443", &tls.Config{ServerName: "api.anthropic.com", MinVersion: tls.VersionTLS12})
	if err != nil {
		p.log("direct Anthropic upgrade dial failed " + in.URL.RequestURI() + ": " + err.Error())
		return writeHTTPResponse(client, http.StatusBadGateway, "application/json", []byte(`{"error":{"type":"api_error","message":"Anthropic Remote Control websocket unavailable"}}`))
	}
	defer upstream.Close()

	out := in.Clone(context.Background())
	out.URL.Scheme = "https"
	out.URL.Host = "api.anthropic.com"
	out.Host = "api.anthropic.com"
	out.RequestURI = ""
	if err := out.Write(upstream); err != nil {
		return err
	}

	done := make(chan struct{}, 2)
	go proxyCopy(done, upstream, client)
	go proxyCopy(done, client, upstream)
	<-done
	return nil
}

var oauthProxyForwardClient = &http.Client{
	Transport: &http.Transport{Proxy: nil},
}

type oauthProxyRoute int

const (
	oauthProxyRouteNone oauthProxyRoute = iota
	oauthProxyRouteLocal
	oauthProxyRouteAnthropic
)

func oauthProxyRouteAllowed(method string, path string) bool {
	return oauthProxyRouteFor(method, path) == oauthProxyRouteLocal
}

func oauthProxyRouteFor(method string, path string) oauthProxyRoute {
	path = normalizeOAuthProxyPath(path)
	switch path {
	case "/api/oauth/usage",
		"/api/oauth/profile",
		"/api/claude_cli_profile",
		"/api/claude_cli/bootstrap",
		"/api/claude_code/settings",
		"/api/claude_code/policy_limits",
		"/api/claude_code_penguin_mode",
		"/v1",
		"/v1/models",
		"/v1/mcp_servers":
		if method == http.MethodGet || method == http.MethodHead {
			return oauthProxyRouteLocal
		}
		return oauthProxyRouteNone
	case "/v1/messages",
		"/v1/messages/count_tokens",
		"/v1/messages/batches":
		if method == http.MethodPost {
			return oauthProxyRouteLocal
		}
		return oauthProxyRouteNone
	default:
		if oauthProxyRemoteControlRouteAllowed(method, path) {
			return oauthProxyRouteAnthropic
		}
		return oauthProxyRouteNone
	}
}

func oauthProxyRemoteControlRouteAllowed(method string, path string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	switch {
	case path == "/v1/sessions" || strings.HasPrefix(path, "/v1/sessions/"):
		return true
	case path == "/v1/code/sessions" || strings.HasPrefix(path, "/v1/code/sessions/"):
		return true
	case path == "/v1/environments" || strings.HasPrefix(path, "/v1/environments/"):
		return true
	case strings.HasPrefix(path, "/v1/session_ingress/"):
		return true
	case methodAllowsReadOnly(method) && strings.HasPrefix(path, "/api/oauth/files/"):
		return true
	default:
		return false
	}
}

func methodAllowsReadOnly(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func isUpgradeRequest(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket") || strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

func normalizeOAuthProxyPath(path string) string {
	for {
		switch {
		case strings.HasPrefix(path, "/v1/v1/"):
			path = "/v1/" + strings.TrimPrefix(path, "/v1/v1/")
		case strings.HasPrefix(path, "/api/v1/"):
			path = "/v1/" + strings.TrimPrefix(path, "/api/v1/")
		default:
			return path
		}
	}
}

func copyForwardHeaders(dst, src http.Header) {
	for key, values := range src {
		if hopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func hopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func writeForwardedResponse(w io.Writer, upstream *http.Response) error {
	resp := &http.Response{
		StatusCode:       upstream.StatusCode,
		Status:           fmt.Sprintf("%d %s", upstream.StatusCode, http.StatusText(upstream.StatusCode)),
		ProtoMajor:       1,
		ProtoMinor:       1,
		Header:           upstream.Header.Clone(),
		Body:             upstream.Body,
		ContentLength:    upstream.ContentLength,
		TransferEncoding: upstream.TransferEncoding,
	}
	for key := range resp.Header {
		if hopByHopHeader(key) {
			resp.Header.Del(key)
		}
	}
	return resp.Write(w)
}

func writeProxyStatus(w io.Writer, status int) error {
	_, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n\r\n", status, http.StatusText(status))
	return err
}

func writeHTTPResponse(w io.Writer, status int, contentType string, body []byte) error {
	var buf bytes.Buffer
	resp := &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
	if contentType != "" {
		resp.Header.Set("content-type", contentType)
	}
	if err := resp.Write(&buf); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func canonicalConnectHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if !strings.Contains(host, ":") {
		return host + ":443"
	}
	return host
}

func generateOAuthProxyCertificate() (tls.Certificate, []byte, error) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	now := time.Now()
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(now.UnixNano()),
		Subject:               pkix.Name{CommonName: "Claudodex Local CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(oauthProxyCertificateLifetime),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano() + 1),
		Subject:      pkix.Name{CommonName: "api.anthropic.com"},
		DNSNames:     []string{"api.anthropic.com"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(oauthProxyCertificateLifetime),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	if caPEM == nil || serverCertPEM == nil || serverKeyPEM == nil {
		return tls.Certificate{}, nil, errors.New("failed to encode OAuth proxy certificate")
	}
	cert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	return cert, caPEM, nil
}
