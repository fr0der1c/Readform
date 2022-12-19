from dataclasses import dataclass


@dataclass
class FeedItem:
    title: str
    url: str
    published_date_parsed: tuple[int, int, int, int, int, int, int, int, int]
