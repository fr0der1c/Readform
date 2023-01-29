import time
import os

from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.expected_conditions import invisibility_of_element_located

from tool_selenium import get_element_with_wait
from website_base import WebsiteAgent
from readwise import send_to_readwise_reader, init_readwise
from driver import get_browser


class TheInitium(WebsiteAgent):
    def __init__(self, driver: WebDriver):
        super().__init__(driver)
        username = os.getenv("THE_INITIUM_USERNAME")
        if not username:
            print("THE_INITIUM_USERNAME not found, cannot proceed")
            exit(1)
        password = os.getenv("THE_INITIUM_PASSWORD")
        if not password:
            print("THE_INITIUM_PASSWORD not found, cannot proceed")
            exit(1)
        self.username = username
        self.password = password

        self.base_domains = ["theinitium.com"]
        self.require_scrolling = True
        self.enable_rss_refreshing = True
        self.rss_address = "https://theinitium.com/newsfeed/"

    def name(self) -> str:
        return "the_initium"

    def check_finish_loading(self):
        if self.get_driver().current_url.startswith("https://theinitium.com/project/"):
            # special handle for projects
            return
        print("changing to simplified Chinese...")
        language_button_locator = (By.XPATH, '//*[@id="user-panel"]/div[3]/div/div/button[2]')
        simplified_button = get_element_with_wait(self.get_driver(), language_button_locator)

        self.get_driver().execute_script("arguments[0].click();", simplified_button)
        # this method can click invisible button

        time.sleep(1)

        self.wait_article_body()
        self.wait_title()  # body 检测不够。有时会出现body加载完成但是标题没加载出来，导致 Reader 无法判断正确标题的情况

    def wait_article_body(self):
        print("waiting for article body to load...")
        get_element_with_wait(self.get_driver(), (By.CSS_SELECTOR, "div.article__body"))
        print("body check passed")

    def wait_title(self):
        while True:
            if self.get_driver().title != "端传媒 Initium Media":
                break
            time.sleep(1)
            print("waiting for title to change...")
        print("title check passed")

    def ensure_logged_in(self):
        if self.get_driver().current_url.startswith("https://theinitium.com/project/"):
            # special handle for projects
            return
        if self.is_paywalled(self.get_driver()):
            print("is paywalled content and not logged-in")
            self.login(self.get_driver())
        else:
            print("is not paywalled content or already logged in")

    def is_paywalled(self, driver: WebDriver) -> bool:
        try:
            driver.find_element(by=By.CSS_SELECTOR,
                                value="#root > main > div.content__body > div > div.main-content > div.article__body > div.paywall")
            return True
        except Exception as e:
            return False

    def login(self, driver: WebDriver):
        """
        Logs in using user credentials on an article page. Make sure to use throttle tool
        to test if the code will work in different speed of network after each change.
        """
        print("logging in...")
        login_link_selector = "#root > main > div.content__body > div > div.main-content > div.article__body > div.paywall > div.link > button"
        login_link = get_element_with_wait(driver, (By.CSS_SELECTOR, login_link_selector))
        login_link.click()
        login_form = get_element_with_wait(driver, (By.CSS_SELECTOR, "div.auth-form"))

        username_box = login_form.find_element(By.NAME, 'email')
        username_box.send_keys(self.username)
        time.sleep(1)  # this is just to simulate real user

        password_box = login_form.find_element(By.NAME, 'password')
        password_box.send_keys(self.password)
        time.sleep(1)

        next_button = login_form.find_element(By.XPATH, '//button[text()="登入"]')
        next_button.click()

        # wait for login form to disappear
        WebDriverWait(driver, timeout=300).until(invisibility_of_element_located(login_form))
        self.wait_article_body()

        # wait paywall to disappear
        paywall_locator = (By.CSS_SELECTOR, "div.paywall")
        WebDriverWait(driver, timeout=300).until(invisibility_of_element_located(paywall_locator))


if __name__ == '__main__':
    init_readwise()

    url = "https://theinitium.com/article/20221227-international-gold-rush-in-africa-2/"
    agent = TheInitium(get_browser())
    full_html = agent.get_page_content(url)
    print(f"get full html success: {full_html}")
    send_to_readwise_reader(url, full_html, agent.name())
    time.sleep(60)
