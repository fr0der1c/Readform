from dataclasses import dataclass
from typing import Optional

FIELD_TYPE_STR = "str"
FIELD_TYPE_STR_LIST = "str_list"
FIELD_TYPE_M_SELECTION = "multiple_selection"
FIELD_TYPE_S_SELECTION = "single_selection"
FIELD_TYPE_BOOL = "bool"


@dataclass
class Selection:
    value: str
    display_name: str


@dataclass
class ConfMeta:
    config_name: str
    config_description: str
    config_key: str

    # optional fields
    typ: Optional[str] = FIELD_TYPE_STR  # supported: str/multiple_selection
    default_value: Optional[str] = ""  # supported: str
    selections: Optional[list[Selection]] = None
    required: Optional[bool] = False
