import os
from selenium.webdriver.chrome.options import Options

chrome_opt = Options()
chrome_opt.add_argument('--no-sandbox')
# chrome_opt.add_argument('--headless')
if os.getenv("IS_IN_CONTAINER"):
    chrome_opt.add_argument('--headless')
chrome_opt.set_capability('chromeOptions', {'w3c': False})
chrome_opt.set_capability('showChromedriverLog', True)
# chrome_opt.add_argument("user-data-dir=chrome-data")
