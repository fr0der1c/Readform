package main

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"os"
	"time"
)

func GetBrowserCtx() (context.Context, context.CancelFunc) {
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"),
	)
	if os.Getenv("IS_IN_CONTAINER") == "" {
		options = append(options, chromedp.Flag("headless", false))
	}
	options = append(options, chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("remote-debugging-port", "5678"), // use chrome://inspect to connect to remote target and debug
		chromedp.WindowSize(2280, 1020),
		// chromedp.Flag("lang", "zh-CN"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	return allocCtx, cancel
}

// AutomationSuite is a tool set for chromedp for quick development of iWebsiteAgent.
type AutomationSuite struct{}

var browser AutomationSuite

func (s AutomationSuite) GetCurrentURL(ctx context.Context) (currentURL string, err error) {
	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Evaluate(`document.URL`, &currentURL),
	})
	if err != nil {
		return "", err
	}
	return currentURL, nil
}

func (s AutomationSuite) GetCurrentTitle(ctx context.Context) (title string, err error) {
	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Title(&title),
	})
	if err != nil {
		return "", err
	}
	return title, nil
}

func (s AutomationSuite) GetHTML(ctx context.Context) (html string, err error) {
	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.OuterHTML("html", &html),
	})
	if err != nil {
		return "", err
	}
	return html, nil
}

func (s AutomationSuite) IsElementHidden(sel string, isHidden *bool) chromedp.Action {
	return chromedp.EvaluateAsDevTools(fmt.Sprintf(`window.getComputedStyle(document.querySelector('%s')).display === 'none'`, sel), &isHidden)
}

func (s AutomationSuite) WaitUntilDocumentReady() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		for {
			var state string
			err := chromedp.Evaluate(`document.readyState`, &state).Do(ctx)
			if err != nil {
				return err
			}
			if state == "complete" {
				return nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	})
}

func (s AutomationSuite) WaitUntilInvisible(ctx context.Context, selector string) error {
	for {
		var nodes []*cdp.Node
		err := chromedp.Run(ctx,
			chromedp.Nodes(selector, &nodes, chromedp.AtLeast(0)),
		) // make sure exist
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			return nil
		}

		var isHidden bool
		err = chromedp.Run(ctx,
			chromedp.EvaluateAsDevTools(fmt.Sprintf(`window.getComputedStyle(document.querySelector('%s')).display === 'none'`, selector), &isHidden),
		)
		if err != nil {
			return err
		}
		if isHidden {
			return nil
		}
	}
}

func (s AutomationSuite) GetWebElementWithWait(ctx context.Context, selector string) (*cdp.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(selector, &nodes, chromedp.BySearch, chromedp.AtLeast(1)))
	if err != nil {
		return nil, fmt.Errorf("WaitWithTimeoutAndInterval failed: %w", err)
	}

	return nodes[0], nil // 返回第一个匹配的节点
}
