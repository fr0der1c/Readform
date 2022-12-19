from datetime import datetime

from sqlalchemy import create_engine, Column, String, Boolean, DateTime, func, or_
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker

from tool_logging import logger

dsn = 'sqlite:///data/readform.db?check_same_thread=False'
engine = create_engine(dsn)  # , echo=True
Base = declarative_base()


class Article(Base):
    __tablename__ = "article"
    url = Column(String(2048), primary_key=True)
    agent = Column(String(128))
    saved_to_readwise = Column(Boolean, default=False)
    save_time = Column(String(128))
    readwise_resp = Column(String(1024))
    content = Column(String())

    create_time = Column(DateTime, server_default=func.now())
    update_time = Column(DateTime, server_default=func.now(), onupdate=func.now())


Session = sessionmaker(bind=engine)


def ensure_db_schema():
    from alembic.config import Config
    from alembic import command

    def run_migrations(script_location: str, dsn: str) -> None:
        logger.info('Running DB migrations in %r on %r', script_location, dsn)
        alembic_cfg = Config()
        alembic_cfg.set_main_option('script_location', script_location)
        alembic_cfg.set_main_option('sqlalchemy.url', dsn)
        command.upgrade(alembic_cfg, 'head')

    run_migrations('alembic', dsn)


def filter_old_urls(url_list: list[str]) -> list[str]:
    """Return URLs not saved to article table."""
    saved_articles = find_article(url_list)
    unsaved_articles = [item for item in url_list if item not in [item.url for item in saved_articles]]
    return unsaved_articles


def filter_saved_urls(url_list: list[str]) -> list[str]:
    """Return URLs not saved to Readwise."""
    saved_articles = find_article(url_list, only_saved=True)
    unsaved_articles = [item for item in url_list if item not in [item.url for item in saved_articles]]
    return unsaved_articles


def _now_time_str() -> str:
    return datetime.now().strftime('%Y-%m-%d %H:%M:%S')


def add_article(url: str, agent: str, content: str):
    """Add an article to article table."""
    session = Session()
    articles = find_article([url], session=session)
    if len(articles) > 0:
        # exist, update
        articles[0].agent = agent
        articles[0].content = content
        session.commit()
    else:
        # create
        article = Article(url=url, agent=agent, content=content)
        session.add(article)
        session.commit()
    pass


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


def find_article(url_list: list[str] = None, session=None, only_saved=False, only_not_saved=False) -> list[Article]:
    if not session:
        session = Session()
    query = session.query(Article)
    if url_list:
        query = query.filter(Article.url.in_(url_list))
    if only_saved:
        query = query.filter(Article.saved_to_readwise == True)
    if only_not_saved:
        query = query.filter(Article.saved_to_readwise == False)

    return query.all()


if __name__ == '__main__':
    # mark_url_as_saved("https://www.google.com", agent="google", resp="{}")
    print(filter_saved_urls(["https://www.google.com", "https://www.google.com/1"]))
