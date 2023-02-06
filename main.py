import queue
import time
import traceback
from typing import Dict
from threading import Thread

from tool_logging import logger
from readwise import init_readwise, send_to_readwise_reader
from conf import load_conf_from_file, current_conf
from website_base import WebsiteAgent
from website_the_initium import TheInitium
from website_caixin import Caixin
from persistence import mark_url_as_saved, ensure_db_schema
from driver import get_browser

handler_dict: Dict[str, WebsiteAgent] = {
}


def init_agents():
    """
    Each website has its own singleton agent instance that owns a Selenium Driver.
    This is to avoid logging into the site again and again.
    """
    websites = current_conf.enabled_websites()
    if not websites:
        logger.error("enabled_websites is empty, please set this env to get the program working")
        exit(1)
    loaded = set()
    for site in websites:
        if site in loaded:
            continue
        if site == "the_initium":
            h = TheInitium(get_browser(), current_conf.get_agent_conf(site))
        elif site == "caixin":
            h = Caixin(get_browser(), current_conf.get_agent_conf(site))
        else:
            logger.error(f"unknown website {site} in READFORM_WEBSITES")
            exit(1)
        for domain in h.base_domains:
            handler_dict[domain] = h
        loaded.add(site)
        time.sleep(1)


class DomainNotSupportedException(Exception):
    pass


def get_page_content(url: str) -> str:
    """inputs a URL and return full HTML"""
    from urllib.parse import urlparse
    domain = urlparse(url).netloc
    domain = '.'.join(domain.split('.')[-2:])  # get base domain
    if domain in handler_dict:
        return handler_dict[domain].get_page_content(url)
    else:
        raise DomainNotSupportedException


article_retry_queue = queue.Queue()


def start_refreshing_rss():
    """Create a refresh thread for each website agent."""
    processed = set()

    def handle_article(single_url: str, agent: str):
        logger.info(f"Getting page content for {single_url}")
        step_name = ""
        try:
            step_name = "getting content"
            content = get_page_content(single_url)

            step_name = "writing local file"
            # write to local to help debug
            with open("data/html/" + single_url.replace("/", "_").replace(":", "") + ".html", "w") as f:
                f.write(content)

            step_name = "sending to Reader"
            logger.info(f"Sending to Readwise Reader...")
            send_to_readwise_reader(single_url, content, agent=agent)
        except Exception as e:
            logger.error(
                f"[{agent}] Got exception while {step_name}: \n{traceback.format_exc()}")
            logger.error(f"Page {single_url} will be retried later.")
            article_retry_queue.put(single_url)

    def refresh_rss(agent: WebsiteAgent):
        """The loop to refresh RSS for a single website."""
        is_first_run = True
        while True:
            logger.info(f"[{agent.name()}] Start to refresh")
            try:
                urls = agent.refresh_rss()
            except Exception as e:
                logger.error(f"[{agent.name()}] Got exception while refreshing RSS: {e.args}\n{traceback.format_exc()}")
                continue
            if len(urls) > 0 and not is_first_run or (is_first_run and current_conf.save_first_fetch()):
                logger.info(f"[{agent.name()}] Latest articles: {urls}")
                for single_url in urls:
                    handle_article(single_url, agent.name())
                    time.sleep(10)
            if is_first_run and not current_conf.save_first_fetch():
                # mark all as saved
                for single_url in urls:
                    mark_url_as_saved(single_url, "_system", "")
                is_first_run = False
            time.sleep(60)
            try:
                url = article_retry_queue.get_nowait()
                if url:
                    handle_article(url, agent.name())
            except queue.Empty:
                pass

    for domain, handler in handler_dict.items():
        if handler not in processed:
            if handler.enable_rss_refreshing:
                t = Thread(target=refresh_rss, args=(handler,), daemon=True)
                t.start()
        else:
            continue
        processed.add(handler)


# todo P1 open a HTTP port to provide RSS feed, so that non-Readwise Reader users can benefit
#         from this project too (add QPS limit to avoid abuse!)
# todo P2 web UI, allows to add source, remove source and suspend fetching, change password on the fly
# todo P2 provide custom feed url parameter

if __name__ == "__main__":
    load_conf_from_file()
    ensure_db_schema()
    init_readwise()
    init_agents()
    start_refreshing_rss()

    while True:
        time.sleep(999)

    # url = "https://theinitium.com/article/20221205-opinion-china-unlock-analysis/"
    # full_html = get_page_content(url)
    # print(f"get full html success: {full_html}")
    #
    # url = "https://theinitium.com/article/20221115-mainland-college-students-crawl/"
    # full_html = get_page_content(url)
    # print(f"get full html success: {full_html}")
