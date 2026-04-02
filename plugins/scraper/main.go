package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"

	tkplugin "github.com/yourorg/toolkit/pkg/plugin"
	"github.com/yourorg/toolkit/pkg/util"
)

// 日志记录
var (
	infoLogger  = log.New(log.Writer(), "[INFO] ", log.Ldate|log.Ltime)
	errorLogger = log.New(log.Writer(), "[ERROR] ", log.Ldate|log.Ltime)
	debugLogger = log.New(log.Writer(), "[DEBUG] ", log.Ldate|log.Ltime)
)

// validateURL 验证 URL 是否安全，防止 SSRF 攻击
func validateURL(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// 仅允许 HTTP/HTTPS
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only HTTP/HTTPS URLs allowed, got: %s", parsed.Scheme)
	}

	// 解析 IP 并检查
	hostname := parsed.Hostname()
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname: %w", err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("access to private IP addresses is forbidden: %s", ip)
		}
	}

	return nil
}

// isPrivateIP 检查是否为内网 IP
func isPrivateIP(ip net.IP) bool {
	// 转换为 IPv4 进行检查
	ip4 := ip.To4()
	if ip4 == nil {
		// IPv6 地址，简单检查回环和链路本地
		return ip.IsLoopback() || ip.IsLinkLocalUnicast()
	}

	// 检查内网 IP 段
	privateRanges := []struct {
		start net.IP
		end   net.IP
	}{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
		{net.ParseIP("127.0.0.0"), net.ParseIP("127.255.255.255")},
		{net.ParseIP("169.254.0.0"), net.ParseIP("169.254.255.255")}, // Link-local
		{net.ParseIP("0.0.0.0"), net.ParseIP("0.255.255.255")},
	}

	for _, r := range privateRanges {
		if ip4.Equal(r.start) || ip4.Equal(r.end) {
			return true
		}
		// 使用字节比较检查范围
		if bytes.Compare(ip4, r.start) >= 0 && bytes.Compare(ip4, r.end) <= 0 {
			return true
		}
	}
	return false
}

// SessionType 会话类型
type SessionType string

const (
	SessionTypeBasic    SessionType = "basic"    // 基本HTTP会话
	SessionTypeStealthy SessionType = "stealthy" // 隐身模式会话
)

// Session 会话接口
type Session interface {
	Get(url string) (*goquery.Document, error)
	Post(url string, data map[string]string) (*goquery.Document, error)
	Close() error
	GetClient() *http.Client
}

// Session 会话实现
type SessionImpl struct {
	client  *http.Client
	stealth bool
	config  *ScraperConfig
}

// Get 发送GET请求
func (s *SessionImpl) Get(url string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// 如果是隐身模式，添加隐身请求头
	if s.stealth {
		s.addStealthHeaders(req)
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	
	return doc, nil
}

// Post 发送POST请求
func (s *SessionImpl) Post(urlStr string, data map[string]string) (*goquery.Document, error) {
	formData := url.Values{}
	for k, v := range data {
		formData.Set(k, v)
	}
	
	req, err := http.NewRequest("POST", urlStr, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// 如果是隐身模式，添加隐身请求头
	if s.stealth {
		s.addStealthHeaders(req)
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	
	return doc, nil
}

// Close 关闭会话
func (s *SessionImpl) Close() error {
	return nil
}

// GetClient 获取HTTP客户端
func (s *SessionImpl) GetClient() *http.Client {
	return s.client
}

// addStealthHeaders 添加隐身请求头
func (s *SessionImpl) addStealthHeaders(req *http.Request) {
	ua := DefaultUserAgents[rand.Intn(len(DefaultUserAgents))]
	
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3")
	req.Header.Set("Accept-Encoding", "gzip, deflate") // 只接受gzip和deflate压缩，不接受brotli压缩
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("TE", "trailers")
	
	// 添加自定义请求头
	for key, value := range s.config.CustomHeaders {
		req.Header.Set(key, value)
	}
}

// ScraperTool 实现爬虫插件 - 基于 Scrapling 设计理念优化
type ScraperTool struct {
	config        *ScraperConfig
	sessions      map[SessionType]Session
	userAgents    []string
	proxyList     []string
	proxyIndex    int
	selectorCache map[string][]SelectorCandidate
	mu            sync.Mutex
}

// ScraperConfig 爬虫配置
type ScraperConfig struct {
	Timeout               int               `json:"timeout" yaml:"timeout"`
	MaxRetries            int               `json:"max_retries" yaml:"max_retries"`
	Delay                 int               `json:"delay" yaml:"delay"`
	RandomDelay           bool              `json:"random_delay" yaml:"random_delay"`
	StealthMode           bool              `json:"stealth_mode" yaml:"stealth_mode"`
	AdaptiveMode          bool              `json:"adaptive_mode" yaml:"adaptive_mode"`
	ProxyList             []string          `json:"proxy_list" yaml:"proxy_list"`
	ProxyRotation         bool              `json:"proxy_rotation" yaml:"proxy_rotation"`
	CustomHeaders         map[string]string `json:"custom_headers" yaml:"custom_headers"`
	FollowRedirects       bool              `json:"follow_redirects" yaml:"follow_redirects"`
	MaxRedirects          int               `json:"max_redirects" yaml:"max_redirects"`
	SkipTLSVerify         bool              `json:"skip_tls_verify" yaml:"skip_tls_verify"`
	ConcurrentLimit       int               `json:"concurrent_limit" yaml:"concurrent_limit"`
	AutoDetectBlocks      bool              `json:"auto_detect_blocks" yaml:"auto_detect_blocks"`
	MaxSelectorCacheSize  int               `json:"max_selector_cache_size" yaml:"max_selector_cache_size"`
}

// SelectorCandidate 选择器候选（用于自适应定位）
type SelectorCandidate struct {
	Selector string
	Score    float64
	Path     string
	Hash     uint64
}

// ElementSignature 元素签名
type ElementSignature struct {
	TextHash   uint64
	Attributes map[string]string
	Position   int
}

// CacheItem 缓存项
type CacheItem struct {
	Key        string
	Value      []SelectorCandidate
	LastAccess time.Time
}

// Cache 缓存管理
type Cache struct {
	items    map[string]*CacheItem
	capacity int
	mu       sync.RWMutex
}

// NewCache 创建新的缓存
func NewCache(capacity int) *Cache {
	return &Cache{
		items:    make(map[string]*CacheItem),
		capacity: capacity,
	}
}

// Get 从缓存中获取数据
func (c *Cache) Get(key string) ([]SelectorCandidate, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	if exists {
		item.LastAccess = time.Now()
	}
	c.mu.RUnlock()

	if exists {
		return item.Value, true
	}
	return nil, false
}

// Set 将数据存入缓存
func (c *Cache) Set(key string, value []SelectorCandidate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果缓存已满，删除最久未使用的项
	if len(c.items) >= c.capacity {
		c.evict()
	}

	c.items[key] = &CacheItem{
		Key:        key,
		Value:      value,
		LastAccess: time.Now(),
	}
}

// evict 删除最久未使用的缓存项
func (c *Cache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.LastAccess
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// Len 返回缓存大小
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// DefaultScraperConfig 默认配置
func DefaultScraperConfig() *ScraperConfig {
	return &ScraperConfig{
		Timeout:               30,
		MaxRetries:            3,
		Delay:                 1,
		RandomDelay:           true,
		StealthMode:           true,
		AdaptiveMode:          true,
		ProxyRotation:         false,
		FollowRedirects:       true,
		MaxRedirects:          10,
		SkipTLSVerify:         true, // 默认为true，跳过证书验证，避免TLS握手失败
		ConcurrentLimit:       5,
		AutoDetectBlocks:      true,
		MaxSelectorCacheSize:  1000,
		CustomHeaders:         make(map[string]string),
	}
}

// DefaultUserAgents 常见的 User-Agent 列表
var DefaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 15_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 15_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
}

// StealthHeaders 隐身模式请求头顺序
var StealthHeaderOrder = []string{
	"User-Agent",
	"Accept",
	"Accept-Language",
	"Accept-Encoding",
	"Connection",
	"Upgrade-Insecure-Requests",
	"Sec-Fetch-Dest",
	"Sec-Fetch-Mode",
	"Sec-Fetch-Site",
	"TE",
}

func (s *ScraperTool) Metadata() tkplugin.ToolMetadata {
	return tkplugin.ToolMetadata{
		Name:        "scraper",
		Version:     "2.0.0",
		Description: "自适应网页爬虫 - 基于 Scrapling 设计，支持智能元素定位、反检测绕过、代理轮换",
		Author:      "Toolkit Team",
		Tags:        []string{"爬虫", "网页", "数据采集", "HTTP", "自适应"},
		Category:    "数据采集",
		ConfigSchema: map[string]tkplugin.Field{
			"timeout": {
				Type:        "integer",
				Description: "请求超时时间（秒）",
				Required:    false,
				Default:     30,
			},
			"max_retries": {
				Type:        "integer",
				Description: "请求失败最大重试次数",
				Required:    false,
				Default:     3,
			},
			"delay": {
				Type:        "integer",
				Description: "请求之间的延迟时间（秒）",
				Required:    false,
				Default:     1,
			},
			"random_delay": {
				Type:        "boolean",
				Description: "启用随机延迟（避免被检测）",
				Required:    false,
				Default:     true,
			},
			"stealth_mode": {
				Type:        "boolean",
				Description: "隐身模式（绕过反机器人检测如 Cloudflare）",
				Required:    false,
				Default:     true,
			},
			"adaptive_mode": {
				Type:        "boolean",
				Description: "自适应模式（元素位置变化时自动重定位）",
				Required:    false,
				Default:     true,
			},
			"proxy_list": {
				Type:        "array",
				Description: "代理服务器列表，格式: http://user:pass@host:port",
				Required:    false,
			},
			"proxy_rotation": {
				Type:        "boolean",
				Description: "启用代理轮换",
				Required:    false,
				Default:     false,
			},
			"follow_redirects": {
				Type:        "boolean",
				Description: "自动跟随重定向",
				Required:    false,
				Default:     true,
			},
			"max_redirects": {
				Type:        "integer",
				Description: "最大重定向次数",
				Required:    false,
				Default:     10,
			},
			"concurrent_limit": {
				Type:        "integer",
				Description: "批量爬取时的最大并发数",
				Required:    false,
				Default:     5,
			},
			"auto_detect_blocks": {
				Type:        "boolean",
				Description: "自动检测并绕过阻止（如验证码页面）",
				Required:    false,
				Default:     true,
			},
		},
	}
}

func (s *ScraperTool) Init(ctx context.Context, config map[string]interface{}) error {
	cfg := DefaultScraperConfig()

	cfg.Timeout = util.GetConfigValue(config, "timeout", cfg.Timeout, util.ToInt)
	cfg.MaxRetries = util.GetConfigValue(config, "max_retries", cfg.MaxRetries, util.ToInt)
	cfg.Delay = util.GetConfigValue(config, "delay", cfg.Delay, util.ToInt)
	cfg.RandomDelay = util.GetConfigValue(config, "random_delay", cfg.RandomDelay, util.ToBool)
	cfg.StealthMode = util.GetConfigValue(config, "stealth_mode", cfg.StealthMode, util.ToBool)
	cfg.AdaptiveMode = util.GetConfigValue(config, "adaptive_mode", cfg.AdaptiveMode, util.ToBool)
	cfg.FollowRedirects = util.GetConfigValue(config, "follow_redirects", cfg.FollowRedirects, util.ToBool)
	cfg.MaxRedirects = util.GetConfigValue(config, "max_redirects", cfg.MaxRedirects, util.ToInt)
	cfg.SkipTLSVerify = util.GetConfigValue(config, "skip_tls_verify", cfg.SkipTLSVerify, util.ToBool)
	cfg.ConcurrentLimit = util.GetConfigValue(config, "concurrent_limit", cfg.ConcurrentLimit, util.ToInt)
	cfg.ProxyRotation = util.GetConfigValue(config, "proxy_rotation", cfg.ProxyRotation, util.ToBool)
	cfg.AutoDetectBlocks = util.GetConfigValue(config, "auto_detect_blocks", cfg.AutoDetectBlocks, util.ToBool)
	cfg.MaxSelectorCacheSize = util.GetConfigValue(config, "max_selector_cache_size", cfg.MaxSelectorCacheSize, util.ToInt)

	if proxyList := util.ToStringSlice(config["proxy_list"]); len(proxyList) > 0 {
		cfg.ProxyList = proxyList
	}
	if customHeaders := util.ToStringMap(config["custom_headers"]); customHeaders != nil {
		cfg.CustomHeaders = customHeaders
	}

	s.config = cfg
	s.userAgents = DefaultUserAgents
	s.proxyList = cfg.ProxyList
	s.selectorCache = make(map[string][]SelectorCandidate)
	s.sessions = make(map[SessionType]Session)

	if err := s.initSessions(); err != nil {
		return fmt.Errorf("初始化会话失败: %w", err)
	}

	return nil
}

// initSessions 初始化会话
func (s *ScraperTool) initSessions() error {
	// 初始化基本HTTP会话
	basicClient, err := s.createHTTPClient(false)
	if err != nil {
		return err
	}
	s.sessions[SessionTypeBasic] = &SessionImpl{
		client:  basicClient,
		stealth: false,
		config:  s.config,
	}

	// 初始化隐身模式会话
	stealthyClient, err := s.createHTTPClient(true)
	if err != nil {
		return err
	}
	s.sessions[SessionTypeStealthy] = &SessionImpl{
		client:  stealthyClient,
		stealth: true,
		config:  s.config,
	}

	return nil
}

// createHTTPClient 创建HTTP客户端
func (s *ScraperTool) createHTTPClient(stealthy bool) (*http.Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	// 支持的密码套件（包括TLS 1.2和TLS 1.3）
	cipherSuites := []uint16{
		// TLS 1.3
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		// TLS 1.2
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		// 额外的TLS 1.2密码套件
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(s.config.Timeout) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false, // 启用压缩，让http.Client自动处理压缩的响应
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: s.config.SkipTLSVerify, // 硬编码为true，确保跳过证书验证
			CipherSuites:       cipherSuites,
			MinVersion:         tls.VersionTLS12, // 降低最低版本以提高兼容性
			MaxVersion:         tls.VersionTLS13,
			SessionTicketsDisabled: false, // 启用会话票证
			ClientSessionCache: tls.NewLRUClientSessionCache(100), // 启用会话缓存
			CurvePreferences: []tls.CurveID{ // 添加曲线偏好
				tls.CurveP256,
				tls.CurveP384,
				tls.CurveP521,
				tls.X25519,
			},
		},
	}

	if s.config.ProxyRotation && len(s.proxyList) > 0 {
		transport.Proxy = s.getNextProxy
	}

	client := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   time.Duration(s.config.Timeout) * time.Second,
	}

	if !s.config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= s.config.MaxRedirects {
				return fmt.Errorf("已达到最大重定向次数 %d", s.config.MaxRedirects)
			}
			return nil
		}
	}

	return client, nil
}

// getNextProxy 获取下一个代理（轮询）
func (s *ScraperTool) getNextProxy(req *http.Request) (*url.URL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.proxyList) == 0 {
		return nil, nil
	}

	proxy := s.proxyList[s.proxyIndex]
	s.proxyIndex = (s.proxyIndex + 1) % len(s.proxyList)

	return url.Parse(proxy)
}

// addStealthHeaders 添加反检测的请求头（模拟真实浏览器）
func (s *ScraperTool) addStealthHeaders(req *http.Request) {
	ua := DefaultUserAgents[rand.Intn(len(DefaultUserAgents))]
	
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3")
	req.Header.Set("Accept-Encoding", "gzip, deflate") // 只接受gzip和deflate压缩，不接受brotli压缩
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("TE", "trailers")

	// 添加自定义请求头
	for key, value := range s.config.CustomHeaders {
		req.Header.Set(key, value)
	}
}

// isBlockedResponse 检测是否被阻止
func (s *ScraperTool) isBlockedResponse(resp *http.Response, body string) bool {
	if !s.config.AutoDetectBlocks {
		return false
	}

	blockedStatus := map[int]bool{
		403: true,
		429: true,
		503: true,
		451: true,
	}
	if blockedStatus[resp.StatusCode] {
		return true
	}

	blockedPatterns := []string{
		"cloudflare",
		"captcha",
		"robot check",
		"access denied",
		"blocked",
		"please verify",
		"attention required",
		"检查你的浏览器",
		"验证您的请求",
		"访问被拒绝",
	}

	bodyLower := strings.ToLower(body)
	for _, pattern := range blockedPatterns {
		if strings.Contains(bodyLower, pattern) {
			return true
		}
	}

	return false
}

// doRequestWithRetries 带重试和阻止检测的请求
func (s *ScraperTool) doRequestWithRetries(req *http.Request) (*http.Response, string, error) {
	var lastErr error

	for i := 0; i < s.config.MaxRetries; i++ {
		if i > 0 {
			delay := time.Duration(s.config.Delay) * time.Second
			if s.config.RandomDelay {
				delay = time.Duration(float64(delay) * (0.5 + rand.Float64()*1.5))
			}
			time.Sleep(delay)

			if s.config.ProxyRotation && len(s.proxyList) > 0 {
				s.mu.Lock()
				s.proxyIndex = (s.proxyIndex + 1) % len(s.proxyList)
				s.mu.Unlock()
			}
		}

		clonedReq := s.cloneRequest(req)
		if s.config.StealthMode {
			s.addStealthHeaders(clonedReq)
		}

		// 根据配置选择会话类型
		sessionType := SessionTypeBasic
		if s.config.StealthMode {
			sessionType = SessionTypeStealthy
		}

		// 获取会话对应的客户端
		session := s.sessions[sessionType]
		client := session.GetClient()
		if client == nil {
			return nil, "", fmt.Errorf("无法获取会话客户端: %v", sessionType)
		}

		resp, err := client.Do(clonedReq)
		if err != nil {
			lastErr = err
			continue
		}

		// 处理gzip压缩的响应
		var reader io.Reader
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(resp.Body)
			if err != nil {
				resp.Body.Close()
				lastErr = err
				continue
			}
			defer gzipReader.Close()
			reader = gzipReader
		} else {
			reader = resp.Body
		}

		bodyBytes, err := io.ReadAll(reader)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		body := string(bodyBytes)

		if s.isBlockedResponse(resp, body) {
			lastErr = fmt.Errorf("检测到访问阻止 (状态码: %d)", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("HTTP 错误: %d %s", resp.StatusCode, resp.Status)
			continue
		}

		return resp, body, nil
	}

	return nil, "", lastErr
}

// cloneRequest 克隆请求
func (s *ScraperTool) cloneRequest(req *http.Request) *http.Request {
	clone := *req
	clone.Header = make(http.Header)
	for k, v := range req.Header {
		clone.Header[k] = v
	}
	return &clone
}

// adaptivelyFindElement 自适应查找元素（当选择器失败时）
func (s *ScraperTool) adaptivelyFindElement(doc *goquery.Document, originalSelector string) *goquery.Selection {
	result := doc.Find(originalSelector)
	if result.Length() > 0 {
		return result
	}

	if !s.config.AdaptiveMode {
		return result
	}

	candidates := s.findAlternativeSelectors(doc, originalSelector)
	for _, candidate := range candidates {
		if sel := doc.Find(candidate.Selector); sel.Length() > 0 {
			return sel
		}
	}

	return result
}

// findAlternativeSelectors 查找替代选择器
func (s *ScraperTool) findAlternativeSelectors(doc *goquery.Document, originalSelector string) []SelectorCandidate {
	sig := fmt.Sprintf("%d", hashString(originalSelector))

	if cached, ok := s.selectorCache[sig]; ok {
		return cached
	}

	candidates := make([]SelectorCandidate, 0)
	
	doc.Find("*").Each(func(i int, sel *goquery.Selection) {
		id, hasID := sel.Attr("id")
		class, hasClass := sel.Attr("class")
		
		if hasID && id != "" {
			candidates = append(candidates, SelectorCandidate{
				Selector: "#" + id,
				Score:    0.9,
			})
		}
		
		if hasClass && class != "" {
			classes := strings.Fields(class)
			for _, c := range classes {
				candidates = append(candidates, SelectorCandidate{
					Selector: "." + c,
					Score:    0.7,
				})
			}
		}
		
		tagName := goquery.NodeName(sel)
		if tagName != "" {
			candidates = append(candidates, SelectorCandidate{
				Selector: tagName,
				Score:    0.3,
			})
		}
	})

	s.selectorCache[sig] = candidates
	
	// 检查缓存大小，超过限制时删除最旧的缓存项
	if len(s.selectorCache) > s.config.MaxSelectorCacheSize {
		// 遍历并删除一半的缓存项（简单的LRU策略）
		count := 0
		for key := range s.selectorCache {
			delete(s.selectorCache, key)
			count++
			if count >= len(s.selectorCache)/2 {
				break
			}
		}
	}
	
	return candidates
}

// hashString 计算字符串哈希
func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func (s *ScraperTool) Execute(ctx context.Context, params map[string]interface{}) (*tkplugin.Result, error) {
	action, _ := params["action"].(string)
	switch action {
	case "fetch":
		return s.fetch(params)
	case "parse":
		return s.parse(params)
	case "links":
		return s.extractLinks(params)
	case "text":
		return s.extractText(params)
	case "crawl":
		return s.crawl(params)
	case "sitemap":
		return s.sitemap(params)
	case "adaptive_parse":
		return s.adaptiveParse(params)
	case "batch":
		return s.batch(params)
	case "export":
		return s.export(params)
	default:
		return &tkplugin.Result{
			Success: false,
			Error:   fmt.Sprintf("未知操作: %s (支持: fetch, parse, links, text, crawl, sitemap, adaptive_parse, batch, export)", action),
		}, nil
	}
}

// fetch 执行基础 HTTP 请求
func (s *ScraperTool) fetch(params map[string]interface{}) (*tkplugin.Result, error) {
	// 获取 URL 参数
	urlStr, _ := params["url"].(string)
	if urlStr == "" {
		return &tkplugin.Result{Success: false, Error: "请提供要访问的网址 (url 参数)"}, nil
	}

	// 验证 URL 安全性，防止 SSRF 攻击
	if err := validateURL(urlStr); err != nil {
		return &tkplugin.Result{Success: false, Error: "网址安全验证失败: " + err.Error()}, nil
	}

	// 获取 HTTP 方法，默认为 GET
	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}

	// 获取请求体
	var body io.Reader
	if requestBody, ok := params["body"].(string); ok && requestBody != "" {
		body = strings.NewReader(requestBody)
	}

	// 创建请求
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "创建请求失败: " + err.Error()}, nil
	}

	// 添加默认请求头
	for key, value := range s.config.CustomHeaders {
		req.Header.Set(key, value)
	}

	// 添加用户自定义请求头
	if headers := util.ToStringMap(params["headers"]); headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	// 执行请求
	resp, bodyStr, err := s.doRequestWithRetries(req)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "请求失败: " + err.Error()}, nil
	}

	// 处理响应头
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// 获取 cookies
	cookies := make([]map[string]interface{}, 0)
	
	// 根据配置选择会话类型
	sessionType := SessionTypeBasic
	if s.config.StealthMode {
		sessionType = SessionTypeStealthy
	}
	
	// 从对应的会话中获取客户端
	session := s.sessions[sessionType]
	client := session.GetClient()
	if client != nil {
		// 从客户端的Jar中获取cookies
		for _, cookie := range client.Jar.Cookies(req.URL) {
			cookies = append(cookies, map[string]interface{}{
				"name":   cookie.Name,
				"value":  cookie.Value,
				"domain": cookie.Domain,
				"path":   cookie.Path,
			})
		}
	}

	// 尝试解析 JSON 响应
	var parsedBody interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal([]byte(bodyStr), &parsedBody); err == nil {
			// JSON 解析成功
			return &tkplugin.Result{
				Success: true,
				Data: map[string]interface{}{
					"url":          urlStr,
					"status_code":  resp.StatusCode,
					"status":       resp.Status,
					"headers":      headers,
					"body":         bodyStr,
					"parsed_body":  parsedBody,
					"cookies":      cookies,
					"content_type": contentType,
				},
				Metrics: map[string]interface{}{
					"response_size": len(bodyStr),
					"is_json":       true,
				},
			}, nil
		}
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"url":          urlStr,
			"status_code":  resp.StatusCode,
			"status":       resp.Status,
			"headers":      headers,
			"body":         bodyStr,
			"cookies":      cookies,
			"content_type": resp.Header.Get("Content-Type"),
		},
		Metrics: map[string]interface{}{
			"response_size": len(bodyStr),
		},
	}, nil
}

// parse 使用 CSS 选择器解析 HTML
func (s *ScraperTool) parse(params map[string]interface{}) (*tkplugin.Result, error) {
	// 获取 URL 参数
	urlStr, _ := params["url"].(string)
	if urlStr == "" {
		return &tkplugin.Result{Success: false, Error: "请提供要解析的网址 (url 参数)"}, nil
	}

	// 获取选择器参数
	selector, _ := params["selector"].(string)
	if selector == "" {
		return &tkplugin.Result{Success: false, Error: "请提供 CSS 选择器 (selector 参数)"}, nil
	}

	// 验证URL安全性，防止SSRF攻击
	if err := validateURL(urlStr); err != nil {
		return &tkplugin.Result{Success: false, Error: "网址安全验证失败: " + err.Error()}, nil
	}

	// 创建请求
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "创建请求失败: " + err.Error()}, nil
	}

	// 执行请求
	resp, bodyStr, err := s.doRequestWithRetries(req)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "请求失败: " + err.Error()}, nil
	}

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "HTML 解析失败: " + err.Error()}, nil
	}

	// 执行选择器
	results := make([]map[string]interface{}, 0)
	selection := doc.Find(selector)
	selection.Each(func(i int, sel *goquery.Selection) {
		results = append(results, extractElementData(sel, i))
	})

	// 提供选择器使用提示
	tips := ""
	if selection.Length() == 0 {
		tips = "未找到匹配元素，可能的原因：1. 选择器语法错误 2. 元素在iframe中 3. 元素需要JavaScript渲染"
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"url":         urlStr,
			"selector":    selector,
			"matches":     len(results),
			"results":     results,
			"adaptive":    false,
			"status_code": resp.StatusCode,
			"tips":        tips,
		},
	}, nil
}

// adaptiveParse 自适应解析（智能元素定位）
func (s *ScraperTool) adaptiveParse(params map[string]interface{}) (*tkplugin.Result, error) {
	// 获取 URL 参数
	urlStr, _ := params["url"].(string)
	if urlStr == "" {
		return &tkplugin.Result{Success: false, Error: "请提供要解析的网址 (url 参数)"}, nil
	}

	// 获取选择器参数
	selector, _ := params["selector"].(string)
	if selector == "" {
		return &tkplugin.Result{Success: false, Error: "请提供 CSS 选择器 (selector 参数)"}, nil
	}

	// 验证URL安全性，防止SSRF攻击
	if err := validateURL(urlStr); err != nil {
		return &tkplugin.Result{Success: false, Error: "网址安全验证失败: " + err.Error()}, nil
	}

	// 创建请求
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "创建请求失败: " + err.Error()}, nil
	}

	// 执行请求
	resp, bodyStr, err := s.doRequestWithRetries(req)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "请求失败: " + err.Error()}, nil
	}

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "HTML 解析失败: " + err.Error()}, nil
	}

	// 尝试使用原始选择器
	sel := doc.Find(selector)
	usedAdaptive := false
	usedSelector := selector
	adaptiveMessage := ""

	// 如果原始选择器没有找到元素，尝试自适应定位
	if sel.Length() == 0 && s.config.AdaptiveMode {
		adaptiveSel := s.adaptivelyFindElement(doc, selector)
		if adaptiveSel.Length() > 0 {
			sel = adaptiveSel
			usedAdaptive = true
			usedSelector = "自适应定位"
			adaptiveMessage = "原始选择器未找到元素，已自动尝试相似元素"
		} else {
			adaptiveMessage = "原始选择器未找到元素，自适应定位也未找到匹配元素"
		}
	}

	// 提取元素数据
	results := make([]map[string]interface{}, 0)
	sel.Each(func(i int, sel *goquery.Selection) {
		results = append(results, extractElementData(sel, i))
	})

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"url":           urlStr,
			"selector":      selector,
			"used_selector": usedSelector,
			"matches":       len(results),
			"results":       results,
			"adaptive":      usedAdaptive,
			"status_code":   resp.StatusCode,
			"message":       adaptiveMessage,
		},
	}, nil
}

// extractElementData 提取元素数据
func extractElementData(sel *goquery.Selection, index int) map[string]interface{} {
	text := sel.Text()
	html, _ := sel.Html()
	
	data := map[string]interface{}{
		"index": index,
		"text":  strings.TrimSpace(text),
		"html":  html,
	}

	attrMap := make(map[string]string)
	for _, attr := range []string{"href", "src", "id", "class", "title", "alt", "data-*"} {
		if strings.HasSuffix(attr, "-*") {
			prefix := strings.TrimSuffix(attr, "*")
			for _, nodeAttr := range sel.Nodes[0].Attr {
				if strings.HasPrefix(nodeAttr.Key, prefix) {
					attrMap[nodeAttr.Key] = nodeAttr.Val
				}
			}
		} else if val, exists := sel.Attr(attr); exists {
			attrMap[attr] = val
		}
	}
	
	if len(attrMap) > 0 {
		data["attributes"] = attrMap
	}

	return data
}

// convertXPathToCSS 将基本 XPath 转换为 CSS 选择器
func convertXPathToCSS(xpath string) string {
	// 简单的 XPath 到 CSS 选择器转换
	// 支持：//tag, //tag[@id='id'], //tag[@class='class']
	css := xpath
	
	// 替换 // 为空格
	css = strings.ReplaceAll(css, "//", " ")
	
	// 处理 id 选择器
	idRegex := regexp.MustCompile(`\[@id='([^']+)'\]`)
	css = idRegex.ReplaceAllString(css, "#$1")
	
	// 处理 class 选择器
	classRegex := regexp.MustCompile(`\[@class='([^']+)'\]`)
	css = classRegex.ReplaceAllString(css, ".$1")
	
	// 移除前导空格
	css = strings.TrimSpace(css)
	
	return css
}

// xpath 使用 XPath 选择器解析 HTML
func (s *ScraperTool) xpath(params map[string]interface{}) (*tkplugin.Result, error) {
	// 获取 URL 参数
	urlStr, _ := params["url"].(string)
	if urlStr == "" {
		return &tkplugin.Result{Success: false, Error: "请提供要解析的网址 (url 参数)"}, nil
	}

	// 获取 XPath 参数
	xpath, _ := params["xpath"].(string)
	if xpath == "" {
		return &tkplugin.Result{Success: false, Error: "请提供 XPath 表达式 (xpath 参数)"}, nil
	}

	// 验证URL安全性，防止SSRF攻击
	if err := validateURL(urlStr); err != nil {
		return &tkplugin.Result{Success: false, Error: "网址安全验证失败: " + err.Error()}, nil
	}

	// 创建请求
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "创建请求失败: " + err.Error()}, nil
	}

	// 执行请求
	resp, bodyStr, err := s.doRequestWithRetries(req)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "请求失败: " + err.Error()}, nil
	}

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "HTML 解析失败: " + err.Error()}, nil
	}

	// 将基本 XPath 转换为 CSS 选择器
	cssSelector := convertXPathToCSS(xpath)
	
	// 执行选择器
	results := make([]map[string]interface{}, 0)
	selection := doc.Find(cssSelector)
	selection.Each(func(i int, sel *goquery.Selection) {
		results = append(results, extractElementData(sel, i))
	})

	// 提供 XPath 使用提示
	tips := ""
	if selection.Length() == 0 {
		tips = "未找到匹配元素，可能的原因：1. XPath 表达式语法错误 2. 元素在iframe中 3. 元素需要JavaScript渲染 4. XPath 转换为CSS选择器时丢失了部分功能"
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"url":          urlStr,
			"xpath":        xpath,
			"css_selector": cssSelector,
			"matches":      len(results),
			"results":      results,
			"status_code":  resp.StatusCode,
			"tips":         tips,
		},
	}, nil
}

// extractLinks 提取页面所有链接
func (s *ScraperTool) extractLinks(params map[string]interface{}) (*tkplugin.Result, error) {
	// 获取 URL 参数
	urlStr, _ := params["url"].(string)
	if urlStr == "" {
		return &tkplugin.Result{Success: false, Error: "请提供要提取链接的网址 (url 参数)"}, nil
	}

	// 验证URL安全性，防止SSRF攻击
	if err := validateURL(urlStr); err != nil {
		return &tkplugin.Result{Success: false, Error: "网址安全验证失败: " + err.Error()}, nil
	}

	// 创建请求
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "创建请求失败: " + err.Error()}, nil
	}

	// 执行请求
	resp, bodyStr, err := s.doRequestWithRetries(req)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "请求失败: " + err.Error()}, nil
	}

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "HTML 解析失败: " + err.Error()}, nil
	}

	// 解析基础 URL
	baseURL, _ := url.Parse(urlStr)
	links := make([]map[string]interface{}, 0)
	uniqueLinks := make(map[string]bool)

	// 提取所有链接
	doc.Find("a").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || href == "" || href == "#" || strings.HasPrefix(href, "javascript:") {
			return
		}

		// 转换为绝对 URL
		absoluteURL := href
		if parsedURL, err := baseURL.Parse(href); err == nil {
			absoluteURL = parsedURL.String()
		}

		// 去重
		if !uniqueLinks[absoluteURL] {
			uniqueLinks[absoluteURL] = true
			text := strings.TrimSpace(sel.Text())
			
			linkData := map[string]interface{}{
				"text": text,
				"href": absoluteURL,
			}
			
			// 添加额外属性
			if title, exists := sel.Attr("title"); exists {
				linkData["title"] = title
			}
			if rel, exists := sel.Attr("rel"); exists {
				linkData["rel"] = rel
			}
			
			links = append(links, linkData)
		}
	})

	// 提供链接提取提示
	tips := ""
	if len(links) == 0 {
		tips = "未找到链接，可能的原因：1. 页面没有链接 2. 链接在iframe中 3. 链接需要JavaScript渲染"
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"url":         urlStr,
			"total":       len(links),
			"links":       links,
			"status_code": resp.StatusCode,
			"tips":        tips,
		},
	}, nil
}

// extractText 提取页面纯文本
func (s *ScraperTool) extractText(params map[string]interface{}) (*tkplugin.Result, error) {
	// 获取 URL 参数
	urlStr, _ := params["url"].(string)
	if urlStr == "" {
		return &tkplugin.Result{Success: false, Error: "请提供要提取文本的网址 (url 参数)"}, nil
	}

	// 验证URL安全性，防止SSRF攻击
	if err := validateURL(urlStr); err != nil {
		return &tkplugin.Result{Success: false, Error: "网址安全验证失败: " + err.Error()}, nil
	}

	// 创建请求
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "创建请求失败: " + err.Error()}, nil
	}

	// 执行请求
	resp, bodyStr, err := s.doRequestWithRetries(req)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "请求失败: " + err.Error()}, nil
	}

	// 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "HTML 解析失败: " + err.Error()}, nil
	}

	// 移除不需要的元素
	doc.Find("script, style, noscript, iframe, svg").Remove()

	// 提取标题和描述
	title := doc.Find("title").Text()
	metaDesc, _ := doc.Find("meta[name='description']").Attr("content")
	
	// 提取正文
	bodySel := doc.Find("body")
	text := bodySel.Text()
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// 提取标题层级
	headings := make([]map[string]interface{}, 0)
	for i := 1; i <= 6; i++ {
		tag := fmt.Sprintf("h%d", i)
		doc.Find(tag).Each(func(j int, sel *goquery.Selection) {
			headings = append(headings, map[string]interface{}{
				"level": i,
				"text":  strings.TrimSpace(sel.Text()),
			})
		})
	}

	// 计算统计信息
	wordCount := len(strings.Fields(text))
	charCount := len(text)
	headingCount := len(headings)

	// 提供文本提取提示
	tips := ""
	if charCount == 0 {
		tips = "未提取到文本，可能的原因：1. 页面为空 2. 文本在iframe中 3. 文本需要JavaScript渲染"
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"url":           urlStr,
			"title":         strings.TrimSpace(title),
			"description":   metaDesc,
			"text":          text,
			"char_count":    charCount,
			"word_count":    wordCount,
			"headings":      headings,
			"heading_count": headingCount,
			"status_code":   resp.StatusCode,
			"tips":          tips,
		},
		Metrics: map[string]interface{}{
			"extraction_time": time.Since(time.Now()).Milliseconds(),
		},
	}, nil
}

// crawl 批量爬取
func (s *ScraperTool) crawl(params map[string]interface{}) (*tkplugin.Result, error) {
	// 获取 URL 列表
	urls := util.ToStringSlice(params["urls"])
	if len(urls) == 0 {
		return &tkplugin.Result{Success: false, Error: "请提供要爬取的网址列表 (urls 参数)"}, nil
	}

	// 获取并发数限制
	limit := s.config.ConcurrentLimit
	if l := util.ToInt(params["limit"]); l > 0 {
		limit = l
	}

	// 获取爬取类型
	crawlType := util.ToString(params["type"])
	if crawlType == "" {
		crawlType = "fetch" // 默认使用 fetch
	}

	// 获取选择器参数
	selector := util.ToString(params["selector"])
	xpath := util.ToString(params["xpath"])

	// 验证参数
	if (crawlType == "parse" && selector == "") || (crawlType == "xpath" && xpath == "") {
		return &tkplugin.Result{Success: false, Error: fmt.Sprintf("%s 类型需要提供 %s 参数", crawlType, map[string]string{"parse": "selector", "xpath": "xpath"}[crawlType])}, nil
	}

	// 准备并发控制
	semaphore := make(chan struct{}, limit)
	results := make([]map[string]interface{}, len(urls))
	var wg sync.WaitGroup

	// 开始批量爬取
	for i, urlStr := range urls {
		wg.Add(1)
		go func(index int, url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var result *tkplugin.Result
			var err error

			// 根据爬取类型执行不同的操作
			switch crawlType {
			case "fetch":
				result, err = s.fetch(map[string]interface{}{"url": url})
			case "text":
				result, err = s.extractText(map[string]interface{}{"url": url})
			case "links":
				result, err = s.extractLinks(map[string]interface{}{"url": url})
			case "parse":
				result, err = s.parse(map[string]interface{}{"url": url, "selector": selector})
			case "xpath":
				result, err = s.xpath(map[string]interface{}{"url": url, "xpath": xpath})
			default:
				result, err = s.fetch(map[string]interface{}{"url": url})
			}

			if err != nil || !result.Success {
				errMsg := ""
				if err != nil {
					errMsg = err.Error()
				} else {
					errMsg = result.Error
				}
				results[index] = map[string]interface{}{
					"url":     url,
					"success": false,
					"error":   errMsg,
				}
				return
			}

			data := result.Data.(map[string]interface{})
			resultItem := map[string]interface{}{
				"url":     url,
				"success": true,
			}

			// 根据爬取类型添加不同的结果数据
			switch crawlType {
			case "fetch":
				resultItem["status_code"] = data["status_code"]
				resultItem["body_size"] = len(data["body"].(string))
			case "text":
				resultItem["title"] = data["title"]
				resultItem["text"] = data["text"]
				resultItem["char_count"] = data["char_count"]
			case "links":
				resultItem["total"] = data["total"]
				resultItem["links"] = data["links"]
			case "parse":
				resultItem["selector"] = data["selector"]
				resultItem["matches"] = data["matches"]
				resultItem["results"] = data["results"]
			case "xpath":
				resultItem["xpath"] = data["xpath"]
				resultItem["matches"] = data["matches"]
				resultItem["results"] = data["results"]
			}

			results[index] = resultItem
		}(i, urlStr)
	}

	// 等待所有爬取任务完成
	wg.Wait()

	// 统计结果
	successCount := 0
	for _, r := range results {
		if s, ok := r["success"].(bool); ok && s {
			successCount++
		}
	}

	// 提供爬取提示
	tips := ""
	if successCount == 0 {
		tips = "所有爬取任务都失败了，可能的原因：1. 网络问题 2. 网址格式错误 3. 网站有反爬虫措施"
	} else if successCount < len(urls) {
		tips = fmt.Sprintf("部分爬取任务失败，成功 %d 个，失败 %d 个", successCount, len(urls)-successCount)
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"results":    results,
			"total":      len(urls),
			"success":    successCount,
			"failed":     len(urls) - successCount,
			"crawl_type": crawlType,
			"selector":   selector,
			"xpath":      xpath,
			"tips":       tips,
		},
		Metrics: map[string]interface{}{
			"crawl_time": time.Since(time.Now()).Milliseconds(),
			"concurrent": limit,
		},
	}, nil
}

// sitemap 生成站点地图
func (s *ScraperTool) sitemap(params map[string]interface{}) (*tkplugin.Result, error) {
	baseURL, _ := params["url"].(string)
	if baseURL == "" {
		return &tkplugin.Result{Success: false, Error: "url 参数不能为空"}, nil
	}

	// 验证URL安全性，防止SSRF攻击
	if err := validateURL(baseURL); err != nil {
		return &tkplugin.Result{Success: false, Error: err.Error()}, nil
	}

	depth := util.ToInt(params["depth"])
	if depth <= 0 {
		depth = 2
	}

	includeExternal := util.ToBool(params["include_external"])

	visited := make(map[string]bool)
	queue := []string{baseURL}
	allLinks := make([]map[string]interface{}, 0)

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return &tkplugin.Result{Success: false, Error: "URL 解析失败: " + err.Error()}, nil
	}

	for currentDepth := 0; currentDepth < depth; currentDepth++ {
		newQueue := make([]string, 0)

		for _, urlStr := range queue {
			if visited[urlStr] {
				continue
			}
			visited[urlStr] = true

			result, err := s.extractLinks(map[string]interface{}{"url": urlStr})
			if err != nil || !result.Success {
				allLinks = append(allLinks, map[string]interface{}{
					"url":    urlStr,
					"status": "failed",
					"depth":  currentDepth + 1,
				})
				continue
			}

			data := result.Data.(map[string]interface{})
			links := data["links"].([]map[string]interface{})

			allLinks = append(allLinks, map[string]interface{}{
				"url":      urlStr,
				"status":   "success",
				"depth":    currentDepth + 1,
				"outbound": len(links),
			})

			for _, link := range links {
				href := link["href"].(string)
				parsedHref, err := url.Parse(href)
				if err != nil {
					continue
				}

				if includeExternal || parsedHref.Host == parsedBase.Host {
					cleanURL := strings.Split(href, "#")[0]
					cleanURL = strings.TrimSuffix(cleanURL, "/")
					// 验证 URL 安全性
					if err := validateURL(cleanURL); err != nil {
						continue
					}
					if !visited[cleanURL] && !strings.HasPrefix(cleanURL, "mailto:") && !strings.HasPrefix(cleanURL, "tel:") {
						newQueue = append(newQueue, cleanURL)
					}
				}
			}
		}

		queue = newQueue
		if len(queue) == 0 {
			break
		}
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"base_url":  baseURL,
			"pages":     allLinks,
			"total":     len(allLinks),
			"max_depth": depth,
			"domain":    parsedBase.Host,
		},
	}, nil
}

func (s *ScraperTool) Validate(params map[string]interface{}) error {
	action, _ := params["action"].(string)
	if action == "" {
		return fmt.Errorf("action 参数是必须的")
	}

	validActions := map[string]bool{
		"fetch":         true,
		"parse":         true,
		"links":         true,
		"text":          true,
		"crawl":         true,
		"sitemap":       true,
		"adaptive_parse": true,
	}

	if !validActions[action] {
		return fmt.Errorf("无效的操作类型: %s", action)
	}

	return nil
}

func (s *ScraperTool) Shutdown(ctx context.Context) error {
	// 关闭所有会话
	for sessionType, session := range s.sessions {
		if err := session.Close(); err != nil {
			// 记录错误但继续关闭其他会话
			fmt.Printf("关闭会话 %v 时出错: %v\n", sessionType, err)
		}
		
		// 关闭会话的空闲连接
		if client := session.GetClient(); client != nil {
			client.CloseIdleConnections()
		}
	}
	return nil
}

// batch 批量执行操作
func (s *ScraperTool) batch(params map[string]interface{}) (*tkplugin.Result, error) {
	tasks, ok := params["tasks"].([]interface{})
	if !ok {
		return &tkplugin.Result{Success: false, Error: "tasks 参数必须是数组"}, nil
	}
	if len(tasks) == 0 {
		return &tkplugin.Result{Success: false, Error: "tasks 参数不能为空数组"}, nil
	}

	limit := s.config.ConcurrentLimit
	if l := util.ToInt(params["limit"]); l > 0 {
		limit = l
	}

	semaphore := make(chan struct{}, limit)
	results := make([]map[string]interface{}, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			results[i] = map[string]interface{}{
				"task":    task,
				"success": false,
				"error":   "任务格式错误",
			}
			continue
		}

		wg.Add(1)
		go func(index int, t map[string]interface{}) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			action, _ := t["action"].(string)
			if action == "" {
				results[index] = map[string]interface{}{
					"task":    t,
					"success": false,
					"error":   "任务缺少action字段",
				}
				return
			}

			result, err := s.Execute(context.Background(), t)
			if err != nil || !result.Success {
				errMsg := ""
				if err != nil {
					errMsg = err.Error()
				} else {
					errMsg = result.Error
				}
				results[index] = map[string]interface{}{
					"task":    t,
					"success": false,
					"error":   errMsg,
				}
				return
			}

			results[index] = map[string]interface{}{
				"task":    t,
				"success": true,
				"data":    result.Data,
			}
		}(i, taskMap)
	}

	wg.Wait()

	successCount := 0
	for _, r := range results {
		if s, ok := r["success"].(bool); ok && s {
			successCount++
		}
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"results": results,
			"total":   len(tasks),
			"success": successCount,
			"failed":  len(tasks) - successCount,
		},
	}, nil
}

// export 导出结果
func (s *ScraperTool) export(params map[string]interface{}) (*tkplugin.Result, error) {
	data, ok := params["data"].(map[string]interface{})
	if !ok {
		return &tkplugin.Result{Success: false, Error: "data 参数必须是对象"}, nil
	}

	format, _ := params["format"].(string)
	if format == "" {
		format = "json"
	}

	var output string
	var err error

	switch strings.ToLower(format) {
	case "json":
		output, err = s.exportJSON(data)
	case "csv":
		output, err = s.exportCSV(data)
	case "txt":
		output, err = s.exportTXT(data)
	default:
		return &tkplugin.Result{Success: false, Error: "不支持的导出格式: " + format}, nil
	}

	if err != nil {
		return &tkplugin.Result{Success: false, Error: "导出失败: " + err.Error()}, nil
	}

	return &tkplugin.Result{
		Success: true,
		Data: map[string]interface{}{
			"format": format,
			"output": output,
			"size":   len(output),
		},
	}, nil
}

// exportJSON 导出为JSON格式
func (s *ScraperTool) exportJSON(data map[string]interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// exportCSV 导出为CSV格式
func (s *ScraperTool) exportCSV(data map[string]interface{}) (string, error) {
	// 简单实现，仅支持二维数组或对象数组
	var csvBuffer bytes.Buffer
	writer := csv.NewWriter(&csvBuffer)

	// 尝试处理不同类型的数据
	switch v := data["results"].(type) {
	case []map[string]interface{}:
		if len(v) == 0 {
			return "", nil
		}

		// 提取表头
		headers := make([]string, 0)
		headerMap := make(map[string]bool)
		for _, item := range v {
			for key := range item {
				if !headerMap[key] {
					headers = append(headers, key)
					headerMap[key] = true
				}
			}
		}

		// 写入表头
		if err := writer.Write(headers); err != nil {
			return "", err
		}

		// 写入数据
		for _, item := range v {
			row := make([]string, len(headers))
			for i, header := range headers {
				if val, ok := item[header]; ok {
					row[i] = fmt.Sprintf("%v", val)
				}
			}
			if err := writer.Write(row); err != nil {
				return "", err
			}
		}

	case []interface{}:
		if len(v) == 0 {
			return "", nil
		}

		// 提取表头
		headers := make([]string, 0)
		headerMap := make(map[string]bool)
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				for key := range itemMap {
					if !headerMap[key] {
						headers = append(headers, key)
						headerMap[key] = true
					}
				}
			}
		}

		// 写入表头
		if err := writer.Write(headers); err != nil {
			return "", err
		}

		// 写入数据
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				row := make([]string, len(headers))
				for i, header := range headers {
					if val, ok := itemMap[header]; ok {
						row[i] = fmt.Sprintf("%v", val)
					}
				}
				if err := writer.Write(row); err != nil {
					return "", err
				}
			}
		}

	default:
		return "", fmt.Errorf("不支持的数据类型: %T", v)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return csvBuffer.String(), nil
}

// exportTXT 导出为文本格式
func (s *ScraperTool) exportTXT(data map[string]interface{}) (string, error) {
	var buffer bytes.Buffer

	// 简单实现，递归打印数据
	var printData func(interface{}, int)
	printData = func(val interface{}, indent int) {
		tab := strings.Repeat("  ", indent)
		switch v := val.(type) {
		case map[string]interface{}:
			for key, value := range v {
				buffer.WriteString(fmt.Sprintf("%s%s:\n", tab, key))
				printData(value, indent+1)
			}
		case []interface{}:
			for i, item := range v {
				buffer.WriteString(fmt.Sprintf("%s[%d]:\n", tab, i))
				printData(item, indent+1)
			}
		default:
			buffer.WriteString(fmt.Sprintf("%s%v\n", tab, v))
		}
	}

	printData(data, 0)
	return buffer.String(), nil
}

func main() {
	tkplugin.Serve(&ScraperTool{})
}
