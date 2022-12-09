import time
import threading
from abc import ABCMeta, abstractmethod
from selenium.webdriver.chrome.webdriver import WebDriver
from tool_rss import parse_rss_feed
from tool_logging import logger
from persistence import filter_saved_urls


class WebsiteAgent(metaclass=ABCMeta):
    def __init__(self, driver: WebDriver):
        self.driver = driver

        # ----------- Below are control options for a website ------------
        self.base_domains = []

        self.require_scrolling = False
        # If enabled, each page will be scrolled over to let the lazy-loaded image load.

        self.enable_rss_refreshing = False
        # If enabled, rss_address will be used to get the latest articles periodically.
        # If disabled, website updates will not be listened. Instead, you will need to
        # manually call API to save content to Readwise.
        self.rss_address = ""

        self._seen_urls = set()

        self._driver_lock = threading.Lock()

        # todo add rate-limit related controls

    @abstractmethod
    def name(self) -> str:
        """
        Return name of the website.
        """
        pass

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
            logger.info(f"got lock for url {url}")
            self.driver.get(url)
            self.check_finish_loading()
            self.ensure_logged_in()

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
        SCROLL_PAUSE_TIME = 0.3

        # Get scroll height
        last_height = self.driver.execute_script("return document.body.scrollHeight")

        while True:
            # Scroll down to bottom
            self.driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")

            # Wait to load page
            time.sleep(SCROLL_PAUSE_TIME)

            # Calculate new scroll height and compare with last scroll height
            new_height = self.driver.execute_script("return document.body.scrollHeight")
            if new_height == last_height:
                break
            last_height = new_height

    def get_driver(self) -> WebDriver:
        """
        This method gets Driver instance.
        A website has a unique Driver instance, and it will be protected by a lock
        to avoid concurrent operations to headless browser. `get_driver` must be
        used instead of directly using `self.driver`.
        """
        return self.driver

    def refresh_rss(self) -> list[str]:
        """
        Refresh RSS to see if there are new content. A list of new article URLs
        will be returned. This method is called by framework if enable_rss_refreshing=True.
        """
        latest_urls = parse_rss_feed(self.rss_address)
        return filter_saved_urls(latest_urls)
