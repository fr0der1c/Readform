from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.wait import WebDriverWait


def get_element_with_wait(driver: WebDriver, locator) -> WebElement:
    return WebDriverWait(driver, 300, 0.5).until(EC.presence_of_element_located(locator))
