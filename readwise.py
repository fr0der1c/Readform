import time

import requests
import traceback
from threading import Thread

from conf import current_conf
from tool_logging import logger
from persistence import mark_url_as_saved, add_article, find_article


def send_to_readwise_reader(url: str, html_content: str, agent=""):
    """send a URL with its HTML content to Reader Feed section"""
    add_article(url, agent, html_content)


def saver():
    logger.info("[Readwise] Saver started running...")
    while True:
        items = find_article(only_not_saved=True, content_not_empty=True)
        if len(items) == 0:
            time.sleep(5)
            continue

        for item in items:
            i = 0
            success = False
            while i <= 3:
                time.sleep(3)
                try:
                    status_code, content = _send_to_readwise_reader(item.url, item.content)
                    if status_code == 200 or status_code == 201:
                        logger.info(f"[Readwise] {status_code} Save {item.url} success: {content}")
                        success = True
                        mark_url_as_saved(item.url, item.agent, content)
                        break
                    logger.error(f"[Readwise] Got {status_code} error when saving {item.url}, retrying...")
                except Exception as e:
                    logger.error(f"[Readwise] Got exception while getting content: \n{traceback.format_exc()}")
                i += 1
            if not success:
                logger.error(f"[Readwise] save {item.url} failed. Will be retried later.")


def _send_to_readwise_reader(url: str, html_content: str) -> (int, str):
    """send a URL with its HTML content to Reader Feed section"""
    resp = requests.post("https://readwise.io/api/v3/save/", json={"url": url,
                                                                   "html": html_content,
                                                                   "should_clean_html": True,
                                                                   "location": current_conf.get_readwise_reader_location(),
                                                                   "saved_using": "ReadForm"},
                         headers={"Authorization": "Token " + current_conf.get_readwise_token()}
                         )

    return resp.status_code, resp.content.decode('utf-8')


def start_saver_thread():
    t = Thread(target=saver, daemon=True)
    t.start()
    # todo wait for the thread to exit


def init_readwise():
    token = current_conf.get_readwise_token()
    if not token:
        logger.error("readwise token not set or empty")
        exit(1)

    start_saver_thread()
