import time
import os

from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.common.by import By
from selenium.common import NoSuchElementException
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.expected_conditions import invisibility_of_element_located

from driver import get_browser
from tool_selenium import get_element_with_wait
from tool_logging import logger
from website_base import WebsiteAgent
from readwise import send_to_readwise_reader, init_readwise


class Caixin(WebsiteAgent):
    def __init__(self, driver: WebDriver):
        super().__init__(driver)
        username = os.getenv("CAIXIN_USERNAME")
        if not username:
            logger.error("CAIXIN_USERNAME not found, cannot proceed")
            exit(1)
        password = os.getenv("CAIXIN_PASSWORD")
        if not password:
            logger.error("CAIXIN_PASSWORD not found, cannot proceed")
            exit(1)
        self.username = username
        self.password = password

        self.base_domains = ["caixin.com"]
        self.require_scrolling = False
        self.enable_rss_refreshing = True
        self.rss_address = "https://rsshub.app/caixin/latest"

    def name(self) -> str:
        return "caixin"

    def check_finish_loading(self):
        self.wait_article_body()

    def wait_article_body(self):
        logger.info("waiting for article body to load...")
        get_element_with_wait(self.get_driver(), (By.CSS_SELECTOR, "#the_content"))

        # Check if video exist
        # Readwise Reader seems cannot handle this type of video yet. But we still
        # make sure it to exist in HTML for future usage.
        try:
            self.driver.find_element(By.CSS_SELECTOR, 'div.content_video')
            logger.info("video found, wait for it to load...")
            get_element_with_wait(self.driver, (By.CSS_SELECTOR, "div.cx-audio-rep"))
        except NoSuchElementException:
            pass
        logger.info("body loading finished")

    def ensure_logged_in(self):
        if self.is_paywalled(self.get_driver()):
            logger.info("is paywalled content and not logged-in")
            self.login(self.get_driver())
        else:
            logger.info("is not paywalled content or already logged in")

    def is_paywalled(self, driver: WebDriver) -> bool:
        try:
            elem = driver.find_element(by=By.CSS_SELECTOR, value="#chargeWallContent")
            return elem.is_displayed()
        except Exception as e:
            return False

    def login(self, driver: WebDriver):
        """
        Logs in using user credentials on an article page. Make sure to use throttle tool
        to test if the code will work in different speed of network after each change.
        """
        logger.info("logging in...")
        login_link_selector = "#chargeWallContent > div > div.loginContent > a"
        login_link = get_element_with_wait(driver, (By.CSS_SELECTOR, login_link_selector))
        login_link.click()

        computer_icon_selector = "#app > div > section > div > div:nth-child(1) > div > div > span > svg > use"
        computer_icon = get_element_with_wait(driver, (By.CSS_SELECTOR, computer_icon_selector))
        computer_icon.click()

        login_form = get_element_with_wait(driver, (
            By.CSS_SELECTOR, "#app > div > section > div > div:nth-child(1) > div > div"))

        username_box = login_form.find_element(By.NAME, 'mobile')
        username_box.clear()
        username_box.send_keys(self.username)
        time.sleep(1)  # this is just to simulate real user

        password_box = login_form.find_element(By.NAME, 'password')
        password_box.send_keys(self.password)
        time.sleep(1)

        tos_selector = "#app > div > section > div > div.cx-login-argree > label > span > span"
        tos = get_element_with_wait(driver, (By.CSS_SELECTOR, tos_selector))
        tos.click()

        next_button = login_form.find_element(By.CSS_SELECTOR, 'button.login-btn')
        next_button.click()
        logger.info("waiting to be redirected...")

        # wait for login form to disappear
        WebDriverWait(driver, timeout=100).until(invisibility_of_element_located(login_form))
        self.wait_article_body()


if __name__ == '__main__':
    init_readwise()
    caixin = Caixin(get_browser())
    # time.sleep(10)

    # contains video
    # url = "https://international.caixin.com/2022-12-07/101975509.html"
    # full_html = caixin.get_page_content(url)
    # print(f"get full html success: {full_html}")
    # send_to_readwise_reader(url, full_html)
    # save_queue.join()

    url = "http://www.caixin.com/2022-12-08/101975979.html"
    full_html = caixin.get_page_content(url)
    print(f"get full html success: {full_html}")
    send_to_readwise_reader(url, full_html, caixin.name())
    save_queue.join()
    #
    # url = "https://www.caixin.com/2022-12-06/101974926.html"
    # full_html = caixin.get_page_content(url)
    # print(f"get full html success: {full_html}")
    # send_to_readwise_reader(url, full_html)
