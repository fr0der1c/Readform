package main

import "context"

type iWebsiteAgent interface {
	// meta definitions
	Name() string                 // agent name, used internally
	DisplayName() string          // agent name showed to user
	ConfOptions() []ConfMeta      // config options offered to user
	BaseDomains() []string        // base domain list
	TestPage() string             // the first page opened for each agent
	URLPrefixBlockList() []string // URL prefixes that the agent could not handle, but appear in RSS feed
	RSSLinks() []string           // default RSS URL list
	RequireScrolling() bool       // if enabled, will scroll to bottom to ensure images are loaded

	// methods to implement for each site
	Init(conf *AgentConf) error                           // init agent
	CleanURL(url string) (string, error)                  // URL in RSS feed may contain unwanted URL parameters, this method will be called to clean URL
	CheckFinishLoading(ctx context.Context) error         // check if page is fully loaded
	IsArticlePaywalled(ctx context.Context) (bool, error) // check if article is paywalled
	EnsureLoggedIn(ctx context.Context) error             // make sure login status
	EventListener(ctx context.Context) func(ev interface{})
}
