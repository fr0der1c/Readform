package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

var allAgents = []*WebsiteAgent{
	{iWebsiteAgent: &TheInitium{}},
	{iWebsiteAgent: &Caixin{}},
	{iWebsiteAgent: &FT{}},
}

type WebsiteAgent struct {
	iWebsiteAgent

	// initialized when running
	ctx                 context.Context
	allocatorCancelFunc context.CancelFunc
	ctxCancelFunc       context.CancelFunc
	conf                *AgentConf // shared between WebsiteAgent and implementations of iWebsiteAgent
	driverLock          sync.Mutex
	retryChan           chan RetryItem
	closeChan           chan struct{}
}

type RetryItem struct {
	URL          string
	RetriedTimes int
}

func (a *WebsiteAgent) Init(ctx context.Context, cancelFunc context.CancelFunc, conf *AgentConf) (err error) {
	a.ctx, a.ctxCancelFunc = chromedp.NewContext(ctx)
	a.allocatorCancelFunc = cancelFunc
	defer func() {
		if err != nil {
			a.Shutdown()
		}
	}()

	err = a.iWebsiteAgent.Init(conf)
	if err != nil {
		return fmt.Errorf("init agent failed: %w", err)
	}

	// recover cookie state
	cookies, err := a.readCookies()
	if err != nil {
		return fmt.Errorf("readCookies failed: %w", err)
	}
	if a.TestPage() == "" {
		return fmt.Errorf("testPage undefined, please define testPage for %s", a.Name())
	}
	err = chromedp.Run(a.ctx, chromedp.Navigate(a.TestPage()), browser.WaitUntilDocumentReady())
	if err != nil {
		return fmt.Errorf("load test page failed: %w", err)
	}
	if len(cookies) > 0 {
		var actions []chromedp.Action
		for _, cookie := range cookies {
			cookie := cookie // a common pitfall when using `for` loop and closure
			actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
				expire := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
				setCookieParams := &network.SetCookieParams{
					Name:         cookie.Name,
					Value:        cookie.Value,
					Domain:       cookie.Domain,
					Path:         cookie.Path,
					Secure:       cookie.Secure,
					HTTPOnly:     cookie.HTTPOnly,
					SameSite:     cookie.SameSite,
					Expires:      &expire,
					Priority:     cookie.Priority,
					SameParty:    cookie.SameParty,
					SourceScheme: cookie.SourceScheme,
					SourcePort:   cookie.SourcePort,
					PartitionKey: cookie.PartitionKey,
				}
				err := setCookieParams.Do(ctx)
				if err != nil {
					return err
				}
				return nil
			}))
		}

		err := chromedp.Run(a.ctx, actions...)
		if err != nil {
			return fmt.Errorf("set cookie failed: %w", err)
		}
	}

	// set agent conf
	a.conf = conf

	a.retryChan = make(chan RetryItem, 10)
	a.closeChan = make(chan struct{})
	return nil
}

func (a *WebsiteAgent) Shutdown() error {
	// not sure what will happen if not cancelled
	a.closeChan <- struct{}{} // send close signal to rss fresh goroutine
	a.ctxCancelFunc()
	a.allocatorCancelFunc()
	return nil
}

var errURLBlocked = errors.New("this URL is blocked by agent")
var errInvalidSubscription = errors.New("logged in but still paywalled. please make sure you have valid subscription to the website")

// GetPageContent gets content of a URL.
func (a *WebsiteAgent) GetPageContent(url string) (currentURL string, htmlContent string, err error) {
	a.driverLock.Lock()
	defer a.driverLock.Unlock()

	ctx, cancelFunc := chromedp.NewContext(a.ctx) // create a new tab for each GetPageContent call
	defer func() {
		cancelFunc()
	}()

	if listener := a.EventListener(ctx); listener != nil {
		// enables agent to listen to events on page, like page.EventJavascriptDialogOpening
		chromedp.ListenTarget(ctx, listener)
	}

	url, err = a.CleanURL(url)
	if err != nil {
		return "", "", fmt.Errorf("clean URL failed: %w", err)
	}
	logger.Infof("[%s] Getting page content for %s", a.Name(), url)

	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(`about:blank`),
		//network.EmulateNetworkConditions(false, 20, 100*1024, 100*1024),
		chromedp.Sleep(1 * time.Second),
		chromedp.Navigate(url),
		browser.WaitUntilDocumentReady(),
		chromedp.Evaluate(`document.URL`, &currentURL),
	})
	if err != nil {
		return "", "", err
	}

	// if loaded URL hits block list, return err(URL could be changed after loading)
	for _, prefix := range a.URLPrefixBlockList() {
		if strings.HasPrefix(currentURL, prefix) {
			return "", "", errURLBlocked
		}
	}

	logger.Infof("[%s] checking loading finish status", a.Name())
	if err := a.CheckFinishLoading(ctx); err != nil {
		return "", "", fmt.Errorf("CheckFinishLoading failed: %w", err)
	}

	isPaywalled, err := a.IsArticlePaywalled(ctx)
	if err != nil {
		return "", "", fmt.Errorf("check IsArticlePaywalled failed: %w", err)
	}
	if isPaywalled {
		// login
		logger.Infof("[%s] checking logging status", a.Name())
		if err := a.EnsureLoggedIn(ctx); err != nil {
			return "", "", fmt.Errorf("EnsureLoggedIn failed: %w", err)
		}
		afterLoginURL, err := browser.GetCurrentURL(ctx)
		if err != nil {
			return "", "", fmt.Errorf("get url failed: %w", err)
		}
		if currentURL != afterLoginURL {
			err = chromedp.Run(ctx, chromedp.Tasks{
				chromedp.Navigate(currentURL),
				browser.WaitUntilDocumentReady(),
			})
			if err != nil {
				return "", "", fmt.Errorf("failed to navigate to original article: %w", err)
			}
		}
		logger.Infof("[%s] checking loading finish status", a.Name())
		if err := a.CheckFinishLoading(ctx); err != nil {
			return "", "", fmt.Errorf("CheckFinishLoading failed: %w", err)
		}

		isPaywalled, err := a.IsArticlePaywalled(ctx)
		if err != nil {
			return "", "", fmt.Errorf("check IsArticlePaywalled failed: %w", err)
		}
		if isPaywalled {
			return "", "", errInvalidSubscription
		}

		logger.Infof("[%s] save cookies", a.Name())
		if err := a.saveCookies(ctx); err != nil {
			return "", "", fmt.Errorf("saveCookies failed: %w", err)
		}
	}

	if a.RequireScrolling() {
		if err := a.scrollPage(ctx); err != nil {
			return "", "", err
		}
	}

	title, err := browser.GetCurrentTitle(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get page title: %v", err)
	}
	if title == "Unable to connect" {
		return "", "", fmt.Errorf("connection error: %v", title)
	}

	html, err := browser.GetHTML(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get page HTML: %w", err)
	}

	currentURL, err = browser.GetCurrentURL(ctx)
	if err != nil {
		return "", "", fmt.Errorf("get current URL failed: %w", err)
	}
	return currentURL, html, nil
}

const ScrollPauseTime = 300 * time.Millisecond
const ScrollOffset = 200

func (a *WebsiteAgent) scrollPage(ctx context.Context) error {
	for {
		// 向下滚动页面
		err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools(fmt.Sprintf("window.scrollBy(0, %d);", ScrollOffset), nil))
		if err != nil {
			return fmt.Errorf("scrollBy failed: %w", err)
		}

		time.Sleep(ScrollPauseTime)

		// 执行JavaScript以获取页面高度
		var pageHeight float64
		err = chromedp.Run(ctx, chromedp.EvaluateAsDevTools("Math.max(document.body.scrollHeight, document.body.offsetHeight, document.documentElement.clientHeight, document.documentElement.scrollHeight, document.documentElement.offsetHeight)", &pageHeight))
		if err != nil {
			return fmt.Errorf("get height failed: %w", err)
		}

		// 执行JavaScript以获取当前滚动位置
		var pageYOffset float64
		err = chromedp.Run(ctx, chromedp.EvaluateAsDevTools("window.pageYOffset", &pageYOffset))
		if err != nil {
			return fmt.Errorf("get pageYOffset failed: %w", err)
		}

		// 执行JavaScript以获取内部窗口高度
		var windowInnerHeight float64
		err = chromedp.Run(ctx, chromedp.EvaluateAsDevTools("window.innerHeight", &windowInnerHeight))
		if err != nil {
			return fmt.Errorf("get innerHeight failed: %w", err)
		}

		// 判断是否已经滚动到了页面的最底部
		if pageHeight-pageYOffset-windowInnerHeight < float64(ScrollOffset) {
			break
		}
	}

	return nil
}

// refreshRSS get unsaved URLs by fetching RSS feed links.
func (a *WebsiteAgent) refreshRSS() ([]string, error) {
	rssAddresses := a.RSSLinks()
	if len(a.conf.RSSLinks) > 0 {
		// use RSS links from conf to overwrite default links defined by agent
		rssAddresses = a.conf.RSSLinks
	}

	var urls []string
	for _, rssAddress := range rssAddresses {
		feedItems, err := ParseRssFeed(rssAddress)
		if err != nil {
			return nil, fmt.Errorf("ParseRssFeed for URL %s failed: %s", rssAddress, err)
		}
		for _, feedItem := range feedItems {
			if a.containsBlockedKeyword(feedItem.Title) {
				logger.Infof("[%s] article %s is filtered because it hits user block keyword list", a.Name(), feedItem.Title)
				continue
			}
			urls = append(urls, feedItem.Link)
		}
	}
	urls, err := filterOldURLs(urls)
	if err != nil {
		return nil, fmt.Errorf("filterOldURLs failed: %w", err)
	}

	var urlsToSave []string
	for _, articleURL := range urls {
		hitBlockList := false
		for _, prefix := range a.URLPrefixBlockList() {
			if strings.HasPrefix(articleURL, prefix) {
				hitBlockList = true
			}
		}
		if !hitBlockList {
			urlsToSave = append(urlsToSave, articleURL)
		}
	}
	return urlsToSave, nil
}

// HandleArticle saves a URL to local file and DB. It will then be checked by sender thread and send to Readwise.
func (a *WebsiteAgent) HandleArticle(url string) error {
	// if URL hits block list, return err immediately
	for _, prefix := range a.URLPrefixBlockList() {
		if strings.HasPrefix(url, prefix) {
			return errURLBlocked
		}
	}

	currentURL, htmlContent, err := a.GetPageContent(url)
	if err != nil {
		return fmt.Errorf("GetPageContent failed: %w", err)
	}

	// save to local file
	err = saveHTMLToLocalFile(currentURL, htmlContent)
	if err != nil {
		return fmt.Errorf("saveHTMLToLocalFile failed: %w", err)
	}

	// save to Article table
	return addArticle(url, a.Name(), currentURL)
}

func (a *WebsiteAgent) StartRefreshingRSS() {
	go func() {
		isFirstRun := true
		for {
			select {
			case <-a.closeChan:
				return
			default:
				// continue refreshing RSS
			}

			logger.Infof("[%s] start to refresh RSS", a.Name())
			articleURLs, err := a.refreshRSS()
			if err != nil {
				logger.Errorf("[%s] refreshRSS failed: %s", a.Name(), err.Error())
				time.Sleep(10 * time.Second)
				continue
			}
			if len(articleURLs) > 0 && (!isFirstRun || (isFirstRun && currentConf.SaveFirstFetch)) {
				logger.Infof("[%s] latest articles: %v", a.Name(), articleURLs)
				for _, url := range articleURLs {
					err := a.HandleArticle(url)
					if err != nil {
						if errors.Is(err, errURLBlocked) {
							logger.Infof("[%s] URL %s is blocked by agent", a.Name(), url)
							continue
						}
						logger.Errorf("[%s] HandleArticle for URL %s failed: %s", a.Name(), url, err)
						// add to retry queue. if full, drop this article and wait for it
						select {
						case a.retryChan <- RetryItem{
							URL:          url,
							RetriedTimes: 0,
						}:
							logger.Infof("[%s] Added %s to retry queue", a.Name(), url)
						default:
							logger.Infof("[%s] URL %s handle failed and retry queue is full, drop this URL.", a.Name(), url)
						}
						time.Sleep(10 * time.Second)
					}
				}
			}
			if isFirstRun && !currentConf.SaveFirstFetch {
				// mark all as saved
				for _, articleURL := range articleURLs {
					err := markURLAsSaved(articleURL, a.Name(), "")
					if err != nil {
						logger.Errorf("[%s] markURLAsSaved failed: %s", a.Name(), err)
					}
				}
				isFirstRun = false
			}

			logger.Infof("[%s] Finished a round of RSS fetch. Sleep 60s.", a.Name())
			time.Sleep(60 * time.Second)

			// retry items from retry queue after each RSS refresh
			select {
			case retryItem := <-a.retryChan:
				logger.Infof("[%s] Retrying URL %s", a.Name(), retryItem.URL)
				err := a.HandleArticle(retryItem.URL)
				if err != nil {
					if errors.Is(err, errURLBlocked) {
						logger.Infof("[%s] URL %s is blocked by agent", a.Name(), retryItem.URL)
						continue
					}
					retryItem.RetriedTimes++
					logger.Errorf("[%s] Retry HandleArticle for URL %s failed: %s, tried %v times", a.Name(), retryItem.URL, err, retryItem.RetriedTimes)
					if retryItem.RetriedTimes >= 10 {
						logger.Errorf("[%s] URL %s is dropped because 10 times of failed HandleArticle", a.Name(), retryItem.URL)
					} else {
						// add to retry queue
						select {
						case a.retryChan <- retryItem:
							// added to retry queue
						default:
							logger.Infof("[%s] URL %s handle failed and retry queue is full, drop this URL.", a.Name(), retryItem.URL)
						}
					}
				}
			default:
				// no item to retry
			}
		}

	}()
}

const CookiePathPrefix = "data/cookie_"

// saveCookies 将当前会话的所有cookies保存到文件
func (a *WebsiteAgent) saveCookies(ctx context.Context) error {
	var cookies []*network.Cookie

	// 运行任务，获取所有的cookies
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	})
	cookieData, err := json.Marshal(cookies)
	if err != nil {
		return err
	}

	cookieFilePath := CookiePathPrefix + a.Name() + ".json"
	err = os.WriteFile(cookieFilePath, cookieData, 0644)
	return err
}

func (a *WebsiteAgent) readCookies() ([]*network.Cookie, error) {
	var cookies []*network.Cookie
	cookieFilePath := CookiePathPrefix + a.Name() + ".json"

	data, err := os.ReadFile(cookieFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return cookies, nil
		}
		return nil, err
	}

	err = json.Unmarshal(data, &cookies)
	if err != nil {
		return nil, err
	}

	return cookies, nil
}

func (a *WebsiteAgent) getTitleBlockedKeywords() []string {
	kw := make([]string, 0, len(a.conf.TitleBlockKeywords))
	for _, item := range kw {
		kw = append(kw, strings.ToLower(item))
	}
	return kw
}

func (a *WebsiteAgent) containsBlockedKeyword(title string) bool {
	title = strings.ToLower(title)
	kwBlockList := a.getTitleBlockedKeywords()

	// 检查标题是否只包含ASCII字符
	isTitleASCII := true
	for _, r := range title {
		if r > unicode.MaxASCII {
			isTitleASCII = false
			break
		}
	}

	if isTitleASCII {
		// 如果标题只包含ASCII字符，拆分单词并检查是否有任何屏蔽关键词
		words := strings.Fields(title)
		wordSet := make(map[string]struct{}, len(words))
		for _, word := range words {
			wordSet[word] = struct{}{}
		}
		for _, keyword := range kwBlockList {
			if _, exists := wordSet[keyword]; exists {
				return true
			}
		}
		return false
	} else {
		// 如果标题包含非ASCII字符，检查标题中是否包含屏蔽关键词
		for _, keyword := range kwBlockList {
			if strings.Contains(title, keyword) {
				return true
			}
		}
		return false
	}
}

// GetAgentForURL 根据 url 获取能处理此url的agent。如果能处理的agent未启用则报错。
func GetAgentForURL(urlStr string) (*WebsiteAgent, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parse url failed: %w", err)
	}

	domain := parsedURL.Hostname()
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		domain = strings.Join(parts[len(parts)-2:], ".")
	}

	if agent, ok := domainAgentDict[domain]; ok {
		return agent, nil
	}

	return nil, fmt.Errorf("supported agent not found for domain %s", domain)
}
