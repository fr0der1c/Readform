import feedparser
from typing import List


def parse_rss_feed(url: str) -> List[str]:
    """Gets a list of item URLs of a RSS feed."""
    feed = feedparser.parse(url)
    urls = []
    for item in feed["entries"]:
        urls.append(item.link)
    return urls
