import json

from tool_logging import logger

CONF_KEY_READWISE_TOKEN = "readwise_token"
CONF_KEY_READER_LOCATION = "reader_location"
CONF_KEY_SAVE_FIRST_FETCH = "save_first_fetch"
CONF_KEY_ENABLED_WEBSITES = "enabled_websites"

GLOBAL_CONFIG_SECTION_NAME = "Global config"

CONF_FILE = "data/conf.json"


class ReadformConf:
    def __init__(self):
        self.dct = {}

    def get_readwise_token(self) -> str:
        return self.dct[CONF_KEY_READWISE_TOKEN] if CONF_KEY_READWISE_TOKEN in self.dct else ""

    def get_readwise_reader_location(self) -> str:
        return self.dct[CONF_KEY_READER_LOCATION] if CONF_KEY_READER_LOCATION in self.dct else "feed"

    def save_first_fetch(self) -> bool:
        return self.dct[CONF_KEY_SAVE_FIRST_FETCH] if CONF_KEY_SAVE_FIRST_FETCH in self.dct else True

    def enabled_websites(self) -> list[str]:
        return self.dct[CONF_KEY_ENABLED_WEBSITES] if CONF_KEY_ENABLED_WEBSITES in self.dct else []

    def get_agent_conf(self, agent_name: str) -> dict:
        if "agent" in self.dct:
            pass
        else:
            self.dct["agent"] = {}
        if agent_name in self.dct["agent"]:
            return self.dct["agent"][agent_name]
        else:
            self.dct["agent"][agent_name] = {}
            return self.dct["agent"][agent_name]

    def set_agent_conf(self, agent_name: str, key, value):
        if "agent" not in self.dct:
            self.dct["agent"] = {}
        if agent_name not in self.dct["agent"]:
            self.dct["agent"][agent_name] = {}
        self.dct["agent"][agent_name][key] = value

    def update_conf(self, new_dct: dict):
        self.dct = new_dct

    def export(self) -> dict:
        from website_base import WebsiteAgent
        from conf_meta import ConfMeta, FIELD_TYPE_M_SELECTION, FIELD_TYPE_S_SELECTION, FIELD_TYPE_BOOL, FIELD_TYPE_STR, Selection
        meta = []
        agent_names = []
        for subclass in WebsiteAgent.__subclasses__():
            meta.append({
                "section": subclass.name,
                "display_name": subclass.display_name,
                "configs": subclass.conf_options
            })
            agent_names.append(Selection(subclass.name, subclass.display_name))
        meta.append({"section": GLOBAL_CONFIG_SECTION_NAME,
                     "configs": [
                         ConfMeta(
                             "Enabled websites",
                             "The websites you are subscribed to. Required.",
                             CONF_KEY_ENABLED_WEBSITES,
                             typ=FIELD_TYPE_M_SELECTION,
                             selections=agent_names,
                             required=True,
                         ),
                         ConfMeta(
                             "Readwise token",
                             "Your Readwise token. Required. Get one at https://readwise.io/access_token",
                             CONF_KEY_READWISE_TOKEN,
                             typ=FIELD_TYPE_STR,
                             required=True,
                         ),
                         ConfMeta(
                             "Readwise Reader location",
                             "The location you would like to save to. Required. Default: feed.",
                             CONF_KEY_READER_LOCATION,
                             typ=FIELD_TYPE_S_SELECTION,
                             selections=[Selection("feed", "Feed"),
                                         Selection("new", "New"),
                                         Selection("later", "Later"),
                                         Selection("archive", "Archive")],
                             required=True,
                         ),
                         ConfMeta(
                             "Save first batch to Readwise Reader",
                             "If to save first batch of articles to Readwise Reader after restarting Readform. Required. Default: Yes.",
                             CONF_KEY_SAVE_FIRST_FETCH,
                             typ=FIELD_TYPE_BOOL,
                             required=True,
                         )
                     ]})
        meta = [meta[-1]] + meta[:-1]  # make global config to be the first

        conf = {}
        for k, v in self.dct.items():
            if k == "agent":
                for agent_name, configs in v.items():
                    for config_k, config_v in configs.items():
                        conf[f"{agent_name}__{config_k}"] = config_v
            else:
                conf[k] = v
        item = {
            "meta": meta,
            "conf": conf
        }
        # print(item)
        return item

    def write_disk(self):
        with open(CONF_FILE, "w") as f:
            f.write(json.dumps(self.dct))


current_conf = ReadformConf()


def load_conf_from_file():
    """load conf from JSON config file"""
    global current_conf
    try:
        with open(CONF_FILE) as f:
            current_conf.update_conf(json.loads(f.read()))
    except FileNotFoundError:
        logger.warning("Config file not found.")
    except json.decoder.JSONDecodeError:
        logger.warning("Config file is not valid JSON, it will be overwritten.")
        # current_conf.update_conf({"agent": {}})
