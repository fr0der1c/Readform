package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/siongui/gojianfan"
	"strings"
	"time"
	"unicode"
)

type TheInitium struct {
	conf *AgentConf

	paywallLocator string
}

func (a *TheInitium) Name() string {
	return "the_initium"
}

func (a *TheInitium) DisplayName() string {
	return "The Initium"
}

func (a *TheInitium) ConfOptions() []ConfMeta {
	return []ConfMeta{
		{
			ConfigName:        "Username",
			ConfigDescription: "Your username for The Initium.",
			ConfigKey:         AgentConfUsername,
			Type:              FieldTypeString,
			Required:          true,
		},
		{
			ConfigName:        "Password",
			ConfigDescription: "Your password for The Initium.",
			ConfigKey:         AgentConfPassword,
			Type:              FieldTypeString,
			Required:          true,
		},
		{
			ConfigName:        "Keyword Blocklist",
			ConfigDescription: "Keywords you want to filter out. Split by comma(,).",
			ConfigKey:         AgentConfKeyBlocklist,
			Type:              FieldTypeStringList,
			Required:          false,
		},
		{
			ConfigName:        "Custom RSS feed link",
			ConfigDescription: "Default feed link is https://rsshub.app/theinitium/channel/latest/zh-hans. You can replace it with your own wanted feed link. Multiple links should split by comma(,).",
			ConfigKey:         AgentConfKeyRSSLinks,
			Type:              FieldTypeStringList,
			Required:          false,
		},
	}
}

func (a *TheInitium) BaseDomains() []string {
	return []string{"theinitium.com"}
}

func (a *TheInitium) TestPage() string {
	return "https://theinitium.com/404"
}

func (a *TheInitium) URLPrefixBlockList() []string {
	return []string{"https://theinitium.com/project/", "https://campaign.theinitium.com/"}
}

func (a *TheInitium) RSSLinks() []string {
	return []string{"https://theinitium.com/newsfeed/"}
}

func (a *TheInitium) RequireScrolling() bool {
	return true
}

func (a *TheInitium) Init(conf *AgentConf) error {
	a.paywallLocator = "//h2[contains(text(), '閱讀全文，歡迎加入會員') or contains(text(), '阅读全文，欢迎加入会员')]"
	a.conf = conf
	return nil
}

func (a *TheInitium) CleanURL(url string) (string, error) {
	return url, nil
}

func (a *TheInitium) CheckFinishLoading(ctx context.Context) error {
	currentURL, err := browser.GetCurrentURL(ctx)
	if err != nil {
		return fmt.Errorf("GetCurrentURL failed: %w", err)
	}

	var languageDetermined, needChangeToSimplifiedChinese bool
	changeToSimplifiedChinese := func() error {

		var nodes []*cdp.Node
		err := chromedp.Run(ctx, chromedp.Nodes(`button[aria-label='繁體中文']`, &nodes, chromedp.AtLeast(0)))
		if err != nil {
			return err
		} else if len(nodes) == 0 {
			logger.Warnf("language switcher not found, ignore switching to simplified Chinese")
			return nil
		}

		logger.Infof("changing to simplified Chinese...")
		// 点击语言按钮
		err = chromedp.Run(ctx,
			chromedp.WaitVisible(`a[aria-label='訂閱支持']`, chromedp.ByQuery),
			chromedp.Click(`a[aria-label='訂閱支持'] + button > span:first-child`, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("failed to click the simplified Chinese option: %w", err)
		}

		time.Sleep(3 * time.Second)
		return nil
	}

	if strings.Contains(currentURL, "zh-Hans") {
		languageDetermined = true
		needChangeToSimplifiedChinese = false
	}
	if !languageDetermined {
		// try to detect by title
		title, err := browser.GetCurrentTitle(ctx)
		if err != nil {
			return fmt.Errorf("get title failed: %w", err)
		}
		if containsTraditionalChinese(title) {
			languageDetermined = true
			needChangeToSimplifiedChinese = true
		}
	}
	if needChangeToSimplifiedChinese {
		err := changeToSimplifiedChinese()
		if err != nil {
			return err
		}
	}

	err = a.WaitArticleBody(ctx)
	if err != nil {
		return err
	}
	err = a.WaitTitle(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a *TheInitium) WaitArticleBody(ctx context.Context) error {
	logger.Infof("waiting for article body to load...")
	_, err := browser.GetWebElementWithWait(ctx, "div[itemprop='articleBody']")
	if err != nil {
		return err
	}
	logger.Infof("body check passed")
	return nil
}

func (a *TheInitium) WaitTitle(ctx context.Context) error {
	for {
		title, err := browser.GetCurrentTitle(ctx)
		if err != nil {
			return err
		}
		if title != "端传媒 Initium Media" {
			break
		}
		time.Sleep(1 * time.Second)
		logger.Infof("waiting for title to change...")
	}
	logger.Infof("title check passed")
	return nil
}

func (a *TheInitium) IsArticlePaywalled(ctx context.Context) (bool, error) {
	return a.isPaywalled(ctx)
}

func (a *TheInitium) EnsureLoggedIn(ctx context.Context) error {
	isPaywalled, err := a.isPaywalled(ctx)
	if err != nil {
		return fmt.Errorf("isPaywalled failed: %w", err)
	}
	if isPaywalled {
		logger.Infof("is paywalled content and not logged-in")

		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.Navigate(`https://theinitium.com/zh-Hans/auth/sign-in`),
			browser.WaitUntilDocumentReady(),
		})
		if err != nil {
			return fmt.Errorf("failed to go to login page: %w", err)
		}

		// 执行登录操作
		err = a.login(ctx)
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}

		return nil
	} else {
		logger.Infof("is not paywalled content or already logged in")
		return nil
	}
}

func (a *TheInitium) isPaywalled(ctx context.Context) (bool, error) {
	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(a.paywallLocator, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return false, err
	} else if len(nodes) == 0 {
		return false, nil
	}
	return true, nil
}

func (a *TheInitium) login(ctx context.Context) error {
	logger.Infof("logging in...")
	username := a.conf.Username
	if username == "" {
		return errors.New("username is empty, cannot proceed")
	}
	password := a.conf.Password
	if password == "" {
		return errors.New("password is empty, cannot proceed")
	}

	err := chromedp.Run(ctx,
		chromedp.WaitVisible(`input[type='email']`),
		chromedp.Focus(`input[type='email']`),
		chromedp.SendKeys(`input[type='email']`, username),
		chromedp.Sleep(1*time.Second),
		chromedp.SendKeys(`input[type='password']`, password),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`button[aria-label='登入']`),
		chromedp.WaitNotPresent(`button[aria-label='登入']`),
	)
	if err != nil {
		return err
	}

	return nil
}

func containsTraditionalChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			// Convert the character to Simplified Chinese
			simplified := gojianfan.T2S(string(r))
			// If the original character is different from the simplified one,
			// it's a Traditional Chinese character.
			if string(r) != simplified {
				return true
			}
		}
	}
	return false
}

func (a *TheInitium) EventListener(ctx context.Context) func(ev interface{}) {
	return nil
}
