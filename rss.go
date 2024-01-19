package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"time"
)

type FeedItem struct {
	Title           string
	Link            string
	PublishedParsed *time.Time
}

// ParseRssFeed 解析RSS源并返回FeedItem列表
func ParseRssFeed(url string) ([]FeedItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("error parsing feed %s: %w", url, err)
	}

	var items []FeedItem
	for _, item := range feed.Items {
		feedItem := FeedItem{
			Title:           item.Title,
			Link:            item.Link,
			PublishedParsed: item.PublishedParsed,
		}
		items = append(items, feedItem)
	}

	return items, nil
}
