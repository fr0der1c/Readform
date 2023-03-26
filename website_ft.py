import time
import random

from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.expected_conditions import invisibility_of_element_located
from selenium.webdriver.common.keys import Keys

from driver import get_browser
from tool_selenium import get_element_with_wait
from tool_logging import logger
from website_base import WebsiteAgent, CONF_KEY_BLOCKLIST, CONF_KEY_RSS_LINKS, MembershipNotValidException
from conf_meta import ConfMeta, FIELD_TYPE_STR_LIST
from conf import current_conf
from readwise import send_to_readwise_reader, init_readwise


class FT(WebsiteAgent):
    name = "financial_times"
    display_name = "Financial Times"  # display name of the website
    conf_options = [
        ConfMeta(
            "FT Username", "Your username for FT.", "username", required=True
        ),
        ConfMeta(
            "RSS feed link",
            "The RSS feed you want to subscribe to. You can generate your personal RSS feed link at https://www.ft.com/myft/alerts/ . For more information, see https://help.ft.com/faq/email-alerts-and-contact-preferences/what-is-myft-rss-feed/",
            CONF_KEY_RSS_LINKS,
            typ=FIELD_TYPE_STR_LIST,
            required=True
        ),
        ConfMeta(
            "Email verification Code",
            "Email verification code will be used to login. Leave this field blank for the first time configuration. After receiving verification email from FT, fill in the code here. This field will be set to black again after login successfully.",
            "access_code", required=False
        ),
        ConfMeta(
            "Keyword Blocklist", "Keywords you want to filter out. Split by comma(,).", CONF_KEY_BLOCKLIST,
            typ=FIELD_TYPE_STR_LIST
        ),
    ]  # all config keys used
    base_domains = ["ft.com"]
    test_page = "https://www.ft.com/404"

    def __init__(self, driver: WebDriver, conf: dict):
        super().__init__(driver, conf)
        self.require_scrolling = False
        self.enable_rss_refreshing = True
        self.rss_addresses = []

    def check_finish_loading(self):
        self.wait_page_load()
        pass

    def wait_page_load(self):
        logger.info("waiting for page to load...")
        get_element_with_wait(self.get_driver(), (By.CSS_SELECTOR, ".o-footer__brand-logo"))
        logger.info("body loading finished")

    def ensure_logged_in(self):
        is_paywalled = self.is_paywalled()
        is_logged_in = self.is_logged_in()
        if is_paywalled and not is_logged_in:
            logger.info("is paywalled content and not logged-in")
            self.login(self.get_driver())
        elif is_logged_in and is_paywalled:
            raise MembershipNotValidException("User is not a valid subscriber of FT. Cannot get full article content.")
        else:
            logger.info(f"Paywall check passed. is_paywalled: {is_paywalled} is_logged_in: {is_logged_in}")

    def is_logged_in(self) -> bool:
        try:
            elem = self.driver.find_element(by=By.CSS_SELECTOR, value="#o-header-top-link-myft")
            return elem.is_displayed()
        except Exception as e:
            logger.warning(f"exception: {e}")
            return False

    def is_paywalled(self) -> bool:
        logger.warning(f"title: {self.driver.title}")
        return self.driver.title.startswith("Subscribe to read")

    def login(self, driver: WebDriver):
        """
        Logs in using user credentials on an article page. Make sure to use throttle tool
        to test if the code will work in different speed of network after each change.
        """
        logger.info("logging in...")
        login_link_selector = "#site-navigation > div.o-header__row.o-header__top > div > div > div.o-header__top-column.o-header__top-column--right > a.o-header__top-link.o-header__top-link--hide-m"
        login_link = get_element_with_wait(driver, (By.CSS_SELECTOR, login_link_selector))
        login_link.click()

        username_box = driver.find_element(By.CSS_SELECTOR, '#enter-email')
        username_box.clear()
        username = self.conf.get("username")
        if not username:
            raise ValueError("username is empty, cannot proceed")
        for char in username:
            username_box.send_keys(char)
            time.sleep(random.uniform(0.3, 0.7))
        username_box.send_keys(Keys.RETURN)

        sign_in_without_password_selector = '#loginWithTokenAnchor'
        sign_in_without_password = get_element_with_wait(driver, (By.CSS_SELECTOR, sign_in_without_password_selector))
        sign_in_without_password.click()

        while True:
            access_code = self.conf.get("access_code")
            if access_code:
                break
            logger.info("Waiting for filling in access code on web UI...")
            time.sleep(10)

        code_selector = ".o-forms-input > input:nth-child(1)"
        code_input = get_element_with_wait(driver, (By.CSS_SELECTOR, code_selector))
        code_input.send_keys(access_code)

        button_selector = "#enter-token-next"
        button = get_element_with_wait(driver, (By.CSS_SELECTOR, button_selector))
        button.click()

        logger.info("waiting to be redirected...")

        # wait for login form to disappear
        WebDriverWait(driver, timeout=100).until(invisibility_of_element_located(button))
        self.wait_page_load()

        current_conf.set_agent_conf(self.name, "access_code", "")
        current_conf.write_disk()


if __name__ == '__main__':
    init_readwise()
    agent = FT(get_browser(), {
        "username": "",
        "password": ""
    })

    url = "https://www.ft.com/content/e5bc3ec2-b522-48c8-880f-7e981c14c9aa"
    full_html = agent.get_page_content(url)
    print(f"get full html success: {full_html}")
    send_to_readwise_reader(url, full_html, agent.name)
