import json

CONF_KEY_READWISE_TOKEN = "readwise_token"
CONF_KEY_READER_LOCATION = "reader_location"
CONF_KEY_SAVE_FIRST_FETCH = "save_first_fetch"
CONF_KEY_ENABLED_WEBSITES = "enabled_websites"


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

    def update_conf(self, new_dct: dict):
        self.dct = new_dct


current_conf = ReadformConf()


def load_conf_from_file():
    """load conf from JSON config file"""
    with open("data/conf.json") as f:
        global current_conf
        current_conf.update_conf(json.loads(f.read()))
