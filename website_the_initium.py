import time
import os

from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.common.by import By
from selenium.common import NoSuchElementException
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.expected_conditions import invisibility_of_element_located

from tool_selenium import get_element_with_wait
from tool_logging import logger
from website_base import WebsiteAgent, CONF_KEY_BLOCKLIST, CONF_KEY_RSS_LINKS
from conf_meta import ConfMeta, FIELD_TYPE_STR_LIST
from readwise import send_to_readwise_reader, init_readwise
from driver import get_browser


class TheInitium(WebsiteAgent):
    name = "the_initium"
    display_name = "The Initium"
    conf_options = [
        ConfMeta(
            "Username", "Your username for The Initium.", "the_initium_username", required=True
        ),
        ConfMeta(
            "Password", "Your password for The Initium.", "the_initium_password", required=True
        ),
        ConfMeta(
            "Keyword Blocklist", "Keywords you want to filter out. Split by comma(,).", CONF_KEY_BLOCKLIST,
            typ=FIELD_TYPE_STR_LIST
        ),
        ConfMeta(
            "Custom RSS feed link",
            "Default feed link is https://rsshub.app/theinitium/channel/latest/zh-hans. You can replace it with your own wanted feed link. Multiple links should split by comma(,).",
            CONF_KEY_RSS_LINKS, typ=FIELD_TYPE_STR_LIST,
        ),
    ]  # all config keys used
    base_domains = ["theinitium.com"]
    test_page = "https://theinitium.com/404"

    def __init__(self, driver: WebDriver, conf: dict):
        super().__init__(driver, conf)
        self.require_scrolling = True
        self.enable_rss_refreshing = True
        self.rss_addresses = ["https://theinitium.com/newsfeed/"]

    def check_finish_loading(self):
        if self.get_driver().current_url.startswith("https://theinitium.com/project/") or \
                self.get_driver().current_url.startswith("https://campaign.theinitium.com/"):
            # special handle for projects
            return

        try:
            logger.info("checking language...")
            self.get_driver().find_element(By.CSS_SELECTOR, "button[aria-label='简体中文']")
        except NoSuchElementException:
            logger.info("changing to simplified Chinese...")
            language_button_locator = (By.CSS_SELECTOR, "button[aria-label='繁體中文']")
            lang_button = get_element_with_wait(self.get_driver(), language_button_locator)
            self.get_driver().execute_script("arguments[0].click();", lang_button)
            # this method can click invisible button

            language_button_locator = (By.XPATH, "//li[contains(text(), '简体中文')]")
            lang_button = get_element_with_wait(self.get_driver(), language_button_locator)
            self.get_driver().execute_script("arguments[0].click();", lang_button)

            time.sleep(3)

        self.wait_article_body()
        self.wait_title()  # body 检测不够。有时会出现body加载完成但是标题没加载出来，导致 Reader 无法判断正确标题的情况

    def wait_article_body(self):
        if self.get_driver().current_url.startswith("https://theinitium.com/project/") or \
                self.get_driver().current_url.startswith("https://campaign.theinitium.com/"):
            # special handle for projects
            return
        logger.info("waiting for article body to load...")
        get_element_with_wait(self.get_driver(), (By.CSS_SELECTOR, "div[itemprop='articleBody']"))
        logger.info("body check passed")

    def wait_title(self):
        while True:
            if self.get_driver().title != "端传媒 Initium Media":
                break
            time.sleep(1)
            logger.info("waiting for title to change...")
        logger.info("title check passed")

    def ensure_logged_in(self):
        if self.get_driver().current_url.startswith("https://theinitium.com/project/") or \
                self.get_driver().current_url.startswith("https://campaign.theinitium.com/"):
            # special handle for projects
            return
        if self.is_paywalled(self.get_driver()):
            logger.info("is paywalled content and not logged-in")
            self.login(self.get_driver())
        else:
            logger.info("is not paywalled content or already logged in")

    PAYWALL_LOCATOR = (By.XPATH,
                       "//h2[contains(text(), '閱讀全文，歡迎加入會員') or contains(text(), '阅读全文，欢迎加入会员')]")

    def is_paywalled(self, driver: WebDriver) -> bool:
        try:
            driver.find_element(*self.PAYWALL_LOCATOR)
            return True
        except Exception as e:
            return False

    def login(self, driver: WebDriver):
        """
        Logs in using user credentials on an article page. Make sure to use throttle tool
        to test if the code will work in different speed of network after each change.
        """
        logger.info("logging in...")

        avatar_selector = "button[aria-label='帐号']"
        avatar = get_element_with_wait(driver, (By.CSS_SELECTOR, avatar_selector))
        avatar.click()

        login_link_selector = "div[data-info='sign-in']"
        login_link = get_element_with_wait(driver, (By.CSS_SELECTOR, login_link_selector))
        login_link.click()

        submit_button = get_element_with_wait(driver, (By.CSS_SELECTOR, "button[type='submit']"))

        username_box = driver.find_element(By.NAME, 'email')
        username = self.conf.get("the_initium_username")
        if not username:
            raise ValueError("the_initium_username is empty, cannot proceed")
        username_box.send_keys(username)
        time.sleep(1)  # this is just to simulate real user

        password_box = driver.find_element(By.NAME, 'password')
        password = self.conf.get("the_initium_password")
        if not password:
            raise ValueError("the_initium_password is empty, cannot proceed")
        password_box.send_keys(password)
        time.sleep(1)

        submit_button.click()

        # wait for login form to disappear
        WebDriverWait(driver, timeout=300).until(invisibility_of_element_located(submit_button))
        self.wait_article_body()

        # wait paywall to disappear
        WebDriverWait(driver, timeout=300).until(invisibility_of_element_located(self.PAYWALL_LOCATOR))


if __name__ == '__main__':
    init_readwise()

    url = "https://theinitium.com/article/20221227-international-gold-rush-in-africa-2/"
    agent = TheInitium(get_browser())
    full_html = agent.get_page_content(url)
    print(f"get full html success: {full_html}")
    send_to_readwise_reader(url, full_html, agent.name())
    time.sleep(60)
