import json
import time
import threading
import queue
import traceback
from typing import Dict, List
from abc import ABCMeta, abstractmethod
from readwise import send_to_readwise_reader
from conf import current_conf
from driver import get_browser
from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.support.wait import WebDriverWait
from selenium.common.exceptions import InvalidSessionIdException
from tool_logging import logger
from tool_rss import parse_rss_feed
from persistence import filter_old_urls, mark_url_as_saved
from selenium.webdriver.common.keys import Keys

CONF_KEY_BLOCKLIST = "title_block_list"
CONF_KEY_RSS_LINKS = "rss_links"

COOKIE_PATH_PREFIX = "data/cookie_"


class MembershipNotValidException(ValueError):
    pass


def contains_chinese(string: str) -> bool:
    for ch in string:
        if u'\u4e00' <= ch <= u'\u9fff':
            return True


class WebsiteAgent(metaclass=ABCMeta):
    name = ""  # name of the website
    display_name = ""  # display name of the website
    conf_options = []  # all config keys used
    base_domains = []
    test_page = ""

    def __init__(self, driver: WebDriver, conf: dict):
        self.driver = driver
        cookies = self.read_cookies()
        if cookies:
            self.driver.get(self.test_page)
            for item in cookies:
                self.driver.add_cookie(item)
        self._driver_lock = threading.Lock()
        self.conf = conf

        # ----------- Below are control options for a website ------------
        self.require_scrolling = False
        # If enabled, each page will be scrolled over to let the lazy-loaded image load.

        self.enable_rss_refreshing = False
        # If enabled, rss_address will be used to get the latest articles periodically.
        # If disabled, website updates will not be listened. Instead, you will need to
        # manually call API to save content to Readwise.
        self.rss_addresses = []
        self._rss_refresh_thread = None

        self.closing = False

        # todo add rate-limit related controls

    @abstractmethod
    def check_finish_loading(self):
        """
        Wait until the loading is finished.
        If the page is in an unexpected status, an error should be raised.
        Do not acquire _driver_lock in this method, as it will cause deadlock.
        """
        pass

    @abstractmethod
    def ensure_logged_in(self):
        """Ensure login status
        Do not acquire _driver_lock in this method, as it will cause deadlock.
        """
        pass

    def get_page_content(self, url: str):
        """
        Inputs a URL and return full HTML. This is the core method and will acquire
        exclusive lock of Selenium driver of current agent.
        """
        with self._driver_lock:
            self.driver.get("about:blank")
            # Set to blank page to avoid elements on last page interfere checks based on element presence.

            time.sleep(1)
            self.driver.get(url)
            # time.sleep(1)  # I've tested, the sleep here is a must.
            wait = WebDriverWait(self.driver, 300)
            wait.until(lambda driver: driver.execute_script("return document.readyState") == "complete")

            self.check_finish_loading()
            self.ensure_logged_in()
            self.check_finish_loading()
            self.save_cookies()

            if self.require_scrolling:
                self.scroll_page()

            # Checking http status code is not possible via vanilla Selenium. Before using other
            # complex ways to detect connection failure, simply checking title will help.
            if self.driver.title in ("Unable to connect",):
                raise ConnectionError(self.driver.title)

            html = self.driver.execute_script("return document.documentElement.outerHTML")
        return html

    def scroll_page(self):
        """Scroll over page til the end."""
        SCROLL_PAUSE_TIME = 0.1

        while True:
            # Scroll down to bottom
            self.driver.execute_script("window.scrollBy(0, 200);")

            # Wait to load page
            time.sleep(SCROLL_PAUSE_TIME)

            js = "return Math.max( document.body.scrollHeight, document.body.offsetHeight,  document.documentElement.clientHeight,  document.documentElement.scrollHeight,  document.documentElement.offsetHeight);"
            page_height = self.driver.execute_script(js)

            # 判断是否已经滚动到了页面的最底部（判断标准是没必要翻下一次）
            if page_height - self.driver.execute_script('return window.pageYOffset;') - self.driver.execute_script(
                    'return window.innerHeight;') < 200:
                break

    def get_driver(self) -> WebDriver:
        """
        This method gets Driver instance.
        A website has a unique Driver instance, and it will be protected by a lock
        to avoid concurrent operations to headless browser. `get_driver` must be
        used instead of directly using `self.driver`.
        """
        return self.driver

    def _get_title_block_list(self) -> list[str]:
        if CONF_KEY_BLOCKLIST in self.conf:
            return [kw.lower() for kw in self.conf[CONF_KEY_BLOCKLIST]]
        return []

    def _contains_blocked_keyword(self, title: str) -> bool:
        blocklist = self._get_title_block_list()
        title = title.lower()
        if title.isascii():
            return not set(title.split(" ")).isdisjoint(blocklist)
        else:
            return any([title.find(kw) >= 0 for kw in blocklist])

    def refresh_rss(self) -> list[str]:
        """
        Refresh RSS to see if there are new content. A list of new article URLs
        will be returned. This method is called by framework if enable_rss_refreshing=True.
        """
        # update self.rss_addresses from user conf
        rss_links_from_conf = self.conf.get(CONF_KEY_RSS_LINKS)
        if rss_links_from_conf is not None and len(
                rss_links_from_conf) > 0 and rss_links_from_conf != self.rss_addresses:
            self.rss_addresses = self.conf.get(CONF_KEY_RSS_LINKS)

        latest_items = []
        for address in self.rss_addresses:
            for item in parse_rss_feed(address):
                if self._contains_blocked_keyword(item.title):
                    logger.info(f"article '{item.title}' is filtered because it hits user block keyword list")
                else:
                    latest_items.append(item)
        return filter_old_urls(latest_items, agent=self.name) if latest_items else []

    def start_refreshing_rss(self):
        t = threading.Thread(target=refresh_rss, args=(self,), daemon=True)
        t.start()

    def close(self):
        self.closing = True  # close rss thread
        self.driver.quit()  # close browser

    def save_cookies(self):
        with open(COOKIE_PATH_PREFIX + self.name, "w") as f:
            f.write(json.dumps(self.driver.get_cookies()))

    def read_cookies(self) -> List[dict]:
        try:
            with open(COOKIE_PATH_PREFIX + self.name, "r") as f:
                return json.loads(f.read())
        except FileNotFoundError:
            return list()
        except json.decoder.JSONDecodeError:
            logger.error(f"cookie file of agent {self.name} is not valid JSON")
            return list()


domain_agent_dict: Dict[str, WebsiteAgent] = {
}


class DomainNotSupportedException(Exception):
    pass


def get_page_content(url: str) -> str:
    """inputs a URL and return full HTML"""
    agent = get_agent_for_url(url)
    return agent.get_page_content(url)


def get_agent_for_url(url: str) -> WebsiteAgent:
    """inputs a URL and return corresponding agent"""
    from urllib.parse import urlparse
    domain = urlparse(url).netloc
    domain = '.'.join(domain.split('.')[-2:])  # get base domain
    if domain in domain_agent_dict:
        return domain_agent_dict[domain]
    else:
        raise DomainNotSupportedException


def handle_article(single_url: str, agent: str, article_retry_queue: queue.Queue):
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
    except InvalidSessionIdException:
        logger.error(f"InvalidSessionIdException detected. try to reopen a browser.")
        get_agent_for_url(single_url).driver.quit()
        get_agent_for_url(single_url).driver = get_browser()


def refresh_rss(agent: WebsiteAgent):
    """The loop to refresh RSS for a single website."""
    article_retry_queue = queue.Queue()
    is_first_run = True
    while True:
        if agent.closing:
            logger.info(f"[{agent.name}] Thread exit")
            return
        logger.info(f"[{agent.name}] Start to refresh")
        try:
            urls = agent.refresh_rss()
        except Exception as e:
            logger.error(f"[{agent.name}] Got exception while refreshing RSS: {e.args}\n{traceback.format_exc()}")
            continue
        if len(urls) > 0 and not is_first_run or (is_first_run and current_conf.save_first_fetch()):
            logger.info(f"[{agent.name}] Latest articles: {urls}")
            for single_url in urls:
                handle_article(single_url, agent.name, article_retry_queue)
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
                handle_article(url, agent.name, article_retry_queue)
        except queue.Empty:
            pass
