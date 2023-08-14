import os
from selenium import webdriver
from selenium.webdriver.firefox.options import Options

firefox_opt = Options()
if os.getenv("IS_IN_CONTAINER"):
    firefox_opt.headless = True
firefox_opt.set_preference("general.useragent.override",
                           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
firefox_opt.set_preference("dom.webdriver.enabled", False)
firefox_opt.set_preference('useAutomationExtension', False)


def get_browser():
    browser = webdriver.Firefox(options=firefox_opt)
    return browser
