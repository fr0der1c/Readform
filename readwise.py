import os
import time
import queue
import requests
import traceback
from threading import Thread

from tool_logging import logger
from persistence import mark_url_as_saved

readwise_token = ""
readwise_location = "feed"

save_queue = queue.Queue()


class Article:
    def __init__(self, url: str, html_content: str, agent: str):
        self.url = url
        self.html_content = html_content
        self.agent = agent


def send_to_readwise_reader(url: str, html_content: str, agent=""):
    """send a URL with its HTML content to Reader Feed section"""
    save_queue.put(Article(url, html_content, agent))


def saver():
    logger.info("[Readwise] Saver started running...")
    while True:
        item = save_queue.get()
        i = 0
        success = False
        while i <= 3:
            time.sleep(3)
            try:
                status_code, content = _send_to_readwise_reader(item.url, item.html_content)
                if status_code == 200 or status_code == 201:
                    logger.info(f"[Readwise] {status_code} Save {item.url} success: {content}")
                    success = True
                    mark_url_as_saved(item.url, item.agent, content)
                    save_queue.task_done()
                    break
                logger.error(f"[Readwise] Got {status_code} error when saving {item.url}, retrying...")
            except Exception as e:
                logger.error(f"[Readwise] Got exception while getting content: \n{traceback.format_exc()}")
            i += 1
        if not success:
            logger.error(f"[Readwise] save {item.url} failed. Add it to queue again for later retry.")
            save_queue.task_done()
            save_queue.put(item)


def _send_to_readwise_reader(url: str, html_content: str) -> (int, str):
    """send a URL with its HTML content to Reader Feed section"""
    resp = requests.post("https://readwise.io/api/v3/save/", json={"url": url,
                                                                   "html": html_content,
                                                                   "should_clean_html": True,
                                                                   "location": readwise_location,
                                                                   "saved_using": "ReadForm"},
                         headers={"Authorization": "Token " + readwise_token}
                         )

    return resp.status_code, resp.content.decode('utf-8')


def start_saver_thread():
    t = Thread(target=saver, daemon=True)
    t.start()
    # todo wait for the thread to exit


def init_readwise():
    token = os.getenv("READWISE_TOKEN")
    if not token:
        logger.error("READWISE_TOKEN not set or empty")
        exit(1)
    global readwise_token
    readwise_token = token

    location = os.getenv("READWISE_READER_LOCATION")
    if location in ("new", "later", "archive", "feed"):
        global readwise_location
        readwise_location = location

    start_saver_thread()
