import feedparser
from typing import List

from datatypes import FeedItem


def parse_rss_feed(url: str) -> List[FeedItem]:
    """Gets a list of FeedItem of an RSS feed."""
    feed = feedparser.parse(url)
    items = []
    for item in feed["entries"]:
        items.append(FeedItem(item.title, item.link, item.published_parsed))
    return items
