import time

from tool_logging import logger
from readwise import init_readwise
from conf import load_conf_from_file, current_conf
from web import run_web_interface
from website_base import domain_agent_dict, WebsiteAgent
from website_caixin import Caixin
from website_the_initium import TheInitium
from persistence import ensure_db_schema
from driver import get_browser


def main_loop():
    """
    Each website has its own singleton agent instance that owns a Selenium Driver.
    This is to avoid logging into the site again and again.
    """
    agents: dict[str:WebsiteAgent] = dict()

    while True:
        enabled_agents = current_conf.enabled_websites()
        if not enabled_agents:
            logger.warning("No enabled website.")

        # start agent for newly enabled websites
        for agent_name in enabled_agents:
            if agent_name in agents.keys():
                continue
            agent = None
            for subclass in WebsiteAgent.__subclasses__():
                if subclass.name == agent_name:
                    agent = subclass(get_browser(), current_conf.get_agent_conf(agent_name))
            if agent is None:
                logger.error(f"Unknown website agent name {agent_name}, ignoring it")
                continue
            logger.info(f"Enabling agent {agent_name}...")
            agents[agent_name] = agent
            for domain in agent.base_domains:
                domain_agent_dict[domain] = agent
            if agent.enable_rss_refreshing:
                agent.start_refreshing_rss()
            time.sleep(1)

        # disable agents for newly disabled websites
        to_del = []
        for agent_name in agents.keys():
            if agent_name in enabled_agents:
                continue
            to_del.append(agent_name)
        for item in to_del:
            logger.info(f"Disabling agent {item}...")
            agents[item].close()
            del agents[item]
        to_del.clear()

        time.sleep(1)


# todo P1 open a HTTP port to provide RSS feed, so that non-Readwise Reader users can benefit
#         from this project too (add QPS limit to avoid abuse!)
# todo P2 provide custom feed url parameter

if __name__ == "__main__":
    load_conf_from_file()
    ensure_db_schema()
    run_web_interface()
    init_readwise()
    main_loop()

    # url = "https://theinitium.com/article/20221205-opinion-china-unlock-analysis/"
    # full_html = get_page_content(url)
    # print(f"get full html success: {full_html}")
    #
    # url = "https://theinitium.com/article/20221115-mainland-college-students-crawl/"
    # full_html = get_page_content(url)
    # print(f"get full html success: {full_html}")
