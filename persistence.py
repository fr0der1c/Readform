from datetime import datetime

from sqlalchemy import create_engine, Column, Integer, String, BOOLEAN
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker

engine = create_engine('sqlite:///data/readform.db?check_same_thread=False')  # , echo=True
Base = declarative_base()


class Article(Base):
    __tablename__ = "article"
    url = Column(String(2048), primary_key=True)
    agent = Column(String(128))
    saved_to_readwise = Column(BOOLEAN)
    save_time = Column(String(128))
    readwise_resp = Column(String(1024))


Base.metadata.create_all(engine, checkfirst=True)
Session = sessionmaker(bind=engine)


def filter_saved_urls(url_list: list[str]) -> list[str]:
    """Return URLs not saved to Readwise."""
    saved_articles = find_article(url_list, only_saved=True)
    unsaved_articles = [item for item in url_list if item not in [item.url for item in saved_articles]]
    return unsaved_articles


def _now_time_str() -> str:
    return datetime.now().strftime('%Y-%m-%d %H:%M:%S')


def mark_url_as_saved(url: str, agent: str, resp: str):
    """Mark a URL as saved to Readwise."""
    session = Session()
    articles = find_article([url], session=session)
    if len(articles) > 0:
        # exist, update
        articles[0].saved_to_readwise = True
        articles[0].save_time = _now_time_str()
        articles[0].agent = agent
        articles[0].readwise_resp = resp
        session.commit()
    else:
        # create
        article = Article(url=url, agent=agent, saved_to_readwise=True, save_time=_now_time_str(), readwise_resp=resp)
        session.add(article)
        session.commit()
    pass


def find_article(url_list: list[str], session=None, only_saved=False) -> list[Article]:
    if not session:
        session = Session()
    query = session.query(Article).filter(Article.url.in_(url_list))
    if only_saved:
        query = query.filter(Article.saved_to_readwise == True)

    return query.all()


if __name__ == '__main__':
    # mark_url_as_saved("https://www.google.com", agent="google", resp="{}")
    print(filter_saved_urls(["https://www.google.com", "https://www.google.com/1"]))
