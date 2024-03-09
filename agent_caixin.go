package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"strings"
	"time"
)

type Caixin struct {
	conf *AgentConf

	paywallLocator string
}

func (a *Caixin) Name() string {
	return "caixin"
}

func (a *Caixin) DisplayName() string {
	return "Caixin"
}

func (a *Caixin) ConfOptions() []ConfMeta {
	return []ConfMeta{
		{
			ConfigName:        "Caixin Username",
			ConfigDescription: "Your username for Caixin.",
			ConfigKey:         AgentConfUsername,
			Type:              FieldTypeString,
			Required:          true,
		},
		{
			ConfigName:        "Caixin Password",
			ConfigDescription: "Your password for Caixin.",
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
			ConfigDescription: "Default feed link is https://rsshub.app/caixin/latest. You can replace it with your own wanted feed link. Multiple links should split by comma(,).",
			ConfigKey:         AgentConfKeyRSSLinks,
			Type:              FieldTypeStringList,
			Required:          false,
		},
		{
			ConfigName:        "Include data-pass(数据通) articles",
			ConfigDescription: "This requires higher level of subscription. Default is false.",
			ConfigKey:         AgentConfIncludePremiumArticles,
			Type:              FieldTypeBool,
			Required:          false,
		},
	}
}

func (a *Caixin) BaseDomains() []string {
	return []string{"caixin.com"}
}

func (a *Caixin) TestPage() string {
	return "https://www.caixin.com/"
}

func (a *Caixin) URLPrefixBlockList() []string {
	prefixList := []string{
		"https://photos.caixin.com",
	}
	if !a.conf.IncludePremiumArticles {
		prefixList = append(prefixList, "https://database.caixin.com")
	}
	return prefixList
}

func (a *Caixin) RSSLinks() []string {
	return []string{"https://rsshub.app/caixin/latest"}
}

func (a *Caixin) RequireScrolling() bool {
	return true
}

func (a *Caixin) Init(conf *AgentConf) error {
	a.paywallLocator = "#chargeWallContent"
	a.conf = conf
	return nil
}

func (a *Caixin) CleanURL(url string) (string, error) {
	return url, nil
}

func (a *Caixin) CheckFinishLoading(ctx context.Context) error {
	logger.Infof("waiting for article body to load...")
	_, err := browser.GetWebElementWithWait(ctx, "#the_content")
	if err != nil {
		return err
	}

	err = browser.WaitUntilInvisible(ctx, "#loadinWall")
	if err != nil {
		return fmt.Errorf("waitUntilInvisible failed: %w", err)
	}

	// Check if video exist
	// Readwise Reader seems cannot handle this type of video yet. But we still make sure it to exist
	// in HTML for future usage.
	var nodes []*cdp.Node
	err = chromedp.Run(ctx, chromedp.Nodes("div.content_video", &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return err
	} else if len(nodes) > 0 {
		logger.Infof("video found, wait for it to load...")
		_, err = browser.GetWebElementWithWait(ctx, "div.cx-audio-rep")
		if err != nil {
			return err
		}
	}
	logger.Infof("body loading finished")

	return nil
}

func (a *Caixin) IsArticlePaywalled(ctx context.Context) (bool, error) {
	return a.isPaywalled(ctx)
}

func (a *Caixin) EnsureLoggedIn(ctx context.Context) error {
	isPaywalled, err := a.isPaywalled(ctx)
	if err != nil {
		return fmt.Errorf("isPaywalled failed: %w", err)
	}
	if isPaywalled {
		logger.Infof("is paywalled content and not logged-in")

		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.Navigate(`https://u.caixin.com/web/login`),
			browser.WaitUntilDocumentReady(),
		})
		if err != nil {
			return fmt.Errorf("failed to go to login page: %w", err)
		}

		currentURL, err := browser.GetCurrentURL(ctx)
		if err != nil {
			return fmt.Errorf("GetCurrentURL failed: %w", err)
		}
		if currentURL == "https://u.caixin.com/web/workbench" {
			// already login
			return fmt.Errorf("already logged in but still paywalled. Check if your subscription is valid")
		}

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

func (a *Caixin) isPaywalled(ctx context.Context) (bool, error) {
	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(a.paywallLocator, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return false, err
	} else if len(nodes) == 0 {
		return false, nil
	} else {
		// check if visible
		var nodes []*cdp.Node
		var isHidden bool
		err := chromedp.Run(ctx,
			chromedp.Nodes(a.paywallLocator, &nodes, chromedp.AtLeast(1)),
			browser.IsElementHidden(a.paywallLocator, &isHidden),
		)
		if err != nil {
			return false, fmt.Errorf("check isHidden failed: %w", err)
		}
		if isHidden {
			return false, nil
		}

		var bodyText string
		err = chromedp.Run(ctx, chromedp.OuterHTML("html", &bodyText))
		if err != nil {
			return false, fmt.Errorf("get document body failed: %w", err)
		}
		if strings.Contains(bodyText, "请升级后阅读") {
			return true, fmt.Errorf("当前用户会员等级不足，需要升级后阅读")
		}
		return true, nil
	}
}

func (a *Caixin) login(ctx context.Context) error {
	logger.Infof("logging in...")
	username := a.conf.Username
	if username == "" {
		return errors.New("username is empty, cannot proceed")
	}
	password := a.conf.Password
	if password == "" {
		return errors.New("password is empty, cannot proceed")
	}
	logger.Infof("next step: waiting icon to be visible")
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(`#app > div > section > div > div:nth-child(1) > div > div > span > svg > use`),
	)
	if err != nil {
		return fmt.Errorf("wait icon visible failed: %w", err)
	}
	err = chromedp.Run(ctx,
		chromedp.Click(`#app > div > section > div > div:nth-child(1) > div > div > span > svg > use`),
	)
	if err != nil {
		return fmt.Errorf("click icon failed: %w", err)
	}
	logger.Infof("next step: wait mobile input to be visible")
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`input[name='mobile']`),
	)
	if err != nil {
		return fmt.Errorf("wait mobilt input visible failed: %w", err)
	}
	err = chromedp.Run(ctx,
		chromedp.Focus(`input[name='mobile']`),
	)
	if err != nil {
		return fmt.Errorf("focus mobile input failed: %w", err)
	}
	logger.Infof("next step: clear mobile input")
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector("input[name='mobile']").value = ""`, nil),
	)
	if err != nil {
		return fmt.Errorf("clear mobile input failed: %w", err)
	}
	logger.Infof("next step: sending mobile number")
	err = chromedp.Run(ctx,
		chromedp.SendKeys(`input[name='mobile']`, username),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		return fmt.Errorf("input mobile number failed: %w", err)
	}
	logger.Infof("next step: send password")
	err = chromedp.Run(ctx,
		chromedp.SendKeys(`input[name='password']`, password),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		return fmt.Errorf("input password failed: %w", err)
	}
	err = chromedp.Run(ctx,
		chromedp.Click(`#app > div > section > div > div.cx-login-argree > label > span > span`),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		return fmt.Errorf("click agreement failed: %w", err)
	}
	logger.Infof("next step: click login button")
	err = chromedp.Run(ctx,
		chromedp.Click(`button.login-btn`),
	)
	if err != nil {
		return fmt.Errorf("click login button failed: %w", err)
	}
	err = chromedp.Run(ctx,
		chromedp.WaitNotPresent(`button.login-btn`),
	)
	if err != nil {
		return fmt.Errorf("wait login button to disappear failed: %w", err)
	}

	return nil
}

func (a *Caixin) EventListener(ctx context.Context) func(ev interface{}) {
	return func(ev interface{}) {
		if ev, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			logger.Warnf("[%s] Reveived dialog message: %+v", a.Name(), ev)
			// 当检测到弹窗时，自动点击确定按钮（当前用于财新被动登出时的弹窗）
			go func() {
				err := chromedp.Run(ctx,
					page.HandleJavaScriptDialog(true),
				)
				if err != nil {
					logger.Errorf("[caixin] HandleJavaScriptDialog failed: %v", err)
				}
			}()
		}
	}
}
