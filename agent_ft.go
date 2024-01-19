package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"net/url"
	"strings"
	"time"
)

type FT struct {
	conf *AgentConf
}

func (a *FT) Name() string {
	return "financial_times"
}

func (a *FT) DisplayName() string {
	return "Financial Times"
}

func (a *FT) ConfOptions() []ConfMeta {
	return []ConfMeta{
		{
			ConfigName:        "FT Username",
			ConfigDescription: "Your username for FT. ",
			ConfigKey:         AgentConfUsername,
			Type:              FieldTypeString,
			Required:          true,
		},
		{
			ConfigName:        "Login method",
			ConfigDescription: "Currently, only email verification code is supported. Please fill in OTP code when there is a OTP prompt window.",
			ConfigKey:         AgentConfLoginMethod,
			Type:              FieldTypeRadio,
			SelectOptions: []SelectOption{
				{Value: "otp_code", DisplayName: "Email verification code"},
			},
			Required:     false,
			DefaultValue: "otp_code",
		},
		{
			ConfigName:        "RSS feed link",
			ConfigDescription: `The RSS feed you want to subscribe to. You can enable your own RSS feed and get feed link at https://www.ft.com/myft/alerts/ under "RSS Feeds" section. For more information, see https://help.ft.com/faq/email-alerts-and-contact-preferences/what-is-myft-rss-feed/`,
			ConfigKey:         AgentConfKeyRSSLinks,
			Type:              FieldTypeStringList,
			Required:          true,
		},
		{
			ConfigName:        "Keyword Blocklist",
			ConfigDescription: "Keywords you want to filter out. Split by comma(,).",
			ConfigKey:         AgentConfKeyBlocklist,
			Type:              FieldTypeStringList,
			Required:          false,
		},
	}
}

func (a *FT) BaseDomains() []string {
	return []string{"ft.com"}
}

func (a *FT) TestPage() string {
	return "https://www.ft.com/"
}

func (a *FT) URLPrefixBlockList() []string {
	return []string{}
}

func (a *FT) RSSLinks() []string {
	return []string{}
}

func (a *FT) RequireScrolling() bool {
	return false
}

func (a *FT) Init(conf *AgentConf) error {
	a.conf = conf
	return nil
}

func (a *FT) closeCookiePopup(ctx context.Context) error {
	const acceptCookiesButtonSelector = `button[title='Accept Cookies']`
	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(acceptCookiesButtonSelector, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return fmt.Errorf("error getting acceptCookiesButton: %w", err)
	} else if len(nodes) == 0 {
		// popup does not exist
		return nil
	}
	err = chromedp.Run(ctx, chromedp.Click(acceptCookiesButtonSelector))
	if err != nil {
		return fmt.Errorf("error clicking acceptCookiesButton: %w", err)
	}
	return nil
}

func (a *FT) CleanURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	return parsedURL.String(), nil
}

func (a *FT) CheckFinishLoading(ctx context.Context) error {
	logger.Infof("waiting for page to load...")

	err := a.closeCookiePopup(ctx)
	if err != nil {
		return fmt.Errorf("closeCookiePopup failed: %w", err)
	}

	_, err = browser.GetWebElementWithWait(ctx, ".o-footer__brand-logo")
	if err != nil {
		return err
	}
	logger.Infof("body loading finished")
	return nil
}

func (a *FT) IsArticlePaywalled(ctx context.Context) (bool, error) {
	return a.isPaywalled(ctx)
}

func (a *FT) EnsureLoggedIn(ctx context.Context) error {
	isPaywalled, err := a.isPaywalled(ctx)
	if err != nil {
		return fmt.Errorf("isPaywalled failed: %w", err)
	}
	isLoggedIn, err := a.isLoggedIn(ctx)
	if err != nil {
		return fmt.Errorf("isLoggedIn failed: %w", err)
	}
	logger.Infof("is_paywalled: %v is_logged_in: %v", isPaywalled, isLoggedIn)
	if isPaywalled && !isLoggedIn {
		logger.Infof("is paywalled content and not logged-in")

		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.Navigate(`https://accounts.ft.com/login`),
			browser.WaitUntilDocumentReady(),
		})
		if err != nil {
			return fmt.Errorf("failed to go to login page: %w", err)
		}

		err = a.login(ctx)
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}
	} else if isLoggedIn && isPaywalled {
		return errors.New("User is not a valid subscriber of FT. Cannot get full article content.")
	}
	return nil
}

func (a *FT) isLoggedIn(ctx context.Context) (bool, error) {
	const myFTIconSelector = "#o-header-top-link-myft"
	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(myFTIconSelector, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return false, fmt.Errorf("find node failed: %w", err)
	} else if len(nodes) == 0 {
		return false, nil
	}
	var isHidden bool
	err = chromedp.Run(ctx, browser.IsElementHidden(myFTIconSelector, &isHidden))
	if err != nil {
		return false, fmt.Errorf("check isHidden failed: %w", err)
	}
	return !isHidden, nil
}

func (a *FT) isPaywalled(ctx context.Context) (bool, error) {
	const paywallIdentSelector = "#barrier-page"
	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(paywallIdentSelector, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return false, err
	} else if len(nodes) == 0 {
		return false, nil
	}
	return true, nil
}

func (a *FT) login(ctx context.Context) error {
	logger.Infof("logging in...")

	err := chromedp.Run(ctx,
		chromedp.WaitVisible(`#enter-email`),
		chromedp.Focus(`#enter-email`),
		chromedp.SendKeys(`#enter-email`, a.conf.Username),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`#enter-email-next`),

		chromedp.WaitVisible(`#loginWithTokenAnchor`),
		chromedp.Click(`#loginWithTokenAnchor`),

		// password login will cause CAPTCHA.
		//chromedp.WaitVisible(`#enter-password`),
		//chromedp.Focus(`#enter-password`),
		//chromedp.SendKeys(`#enter-password`, a.conf.Password),
		//chromedp.Sleep(1*time.Second),
		//
		//chromedp.Click(`#sign-in-button`),
		//chromedp.WaitNotPresent(`#sign-in-button`),
	)
	if err != nil {
		return err
	}

	logger.Infof("Please to go Readform web console to input OTP code")
	otp := RequireOTP(a.DisplayName())
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`input[type='token']`),
		chromedp.SendKeys(`input[type='token']`, otp),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`#enter-token-next`),
		chromedp.Sleep(1*time.Second),
		browser.WaitUntilDocumentReady(),
	)
	if err != nil {
		return err
	}

	html, err := browser.GetHTML(ctx)
	if err != nil {
		return fmt.Errorf("get html failed: %w", err)
	}
	if strings.Contains(html, "Invalid or expired code") {
		return fmt.Errorf("OTP valid or expired. OTP will be requested again when retrying")
	}

	return nil
}

func (a *FT) EventListener(ctx context.Context) func(ev interface{}) {
	return nil
}
