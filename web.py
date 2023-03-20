from threading import Thread
from typing import Optional, Any

from flask import Flask, render_template, request

from conf import current_conf, GLOBAL_CONFIG_SECTION_NAME, CONF_KEY_ENABLED_WEBSITES
from conf_meta import ConfMeta
from tool_logging import logger

app = Flask(__name__)
app.logger = logger


@app.route('/')
def index():
    return render_template('index.html', config_options=current_conf.export())


def check_single_config(lst: list, cfg_meta: ConfMeta) -> (Any, bool):
    from conf_meta import FIELD_TYPE_STR, FIELD_TYPE_STR_LIST, FIELD_TYPE_S_SELECTION, FIELD_TYPE_M_SELECTION, \
        FIELD_TYPE_BOOL
    if cfg_meta.typ == FIELD_TYPE_STR:
        return lst[0], True
    elif cfg_meta.typ == FIELD_TYPE_STR_LIST:
        if lst[0] == "":
            return [], True
        items: list[str] = lst[0].split(",")
        new_items = [item.strip() for item in items if item.strip() != ""]
        return new_items, True
    elif cfg_meta.typ == FIELD_TYPE_S_SELECTION:
        if len(lst) == 0:
            return None, False
        return lst[0], True
    elif cfg_meta.typ == FIELD_TYPE_M_SELECTION:
        return lst, True
    elif cfg_meta.typ == FIELD_TYPE_BOOL:
        return lst[0] == "True", True
    return None, False


def is_value_empty(value) -> bool:
    return value is None or value == "" or value == []


@app.route('/save_config', methods=["POST"])
def save_config():
    if request.method == "POST":
        current_export = current_conf.export()
        meta = current_export["meta"]

        # param check & get enabled agent list
        enabled_agents = request.form.getlist(CONF_KEY_ENABLED_WEBSITES)

        # param check
        for meta_item in meta:
            config_list: list[ConfMeta] = meta_item["configs"]
            if meta_item["section"] == GLOBAL_CONFIG_SECTION_NAME:
                for config in config_list:
                    value_in_form = request.form.getlist(config.config_key)
                    cfg_v, ok = check_single_config(value_in_form, config)
                    if ok:
                        if is_value_empty(cfg_v) and config.required:
                            return {"success": False, "message": f'Config "{config.config_name}" is required.'}
                    else:
                        return {"success": False, "message": f'Config "{config.config_name}" is not valid value.'}
            else:
                agent_name = meta_item["section"]
                for config in config_list:
                    form_key = agent_name + "__" + config.config_key
                    value_in_form = request.form.getlist(form_key)
                    cfg_v, ok = check_single_config(value_in_form, config)
                    if ok:
                        if config.required and agent_name in enabled_agents and is_value_empty(cfg_v):
                            return {"success": False,
                                    "message": f'Config "{config.config_name}" of "{agent_name}" is required.'}
                    else:
                        return {"success": False, "message": f'Config "{config.config_name}" is not valid value.'}

        for meta_item in meta:
            config_list: list[ConfMeta] = meta_item["configs"]
            if meta_item["section"] == GLOBAL_CONFIG_SECTION_NAME:
                for config in config_list:
                    value_in_form = request.form.getlist(config.config_key)
                    cfg_v, _ = check_single_config(value_in_form, config)
                    current_conf.dct[config.config_key] = cfg_v
            else:
                agent_name = meta_item["section"]
                for config in config_list:
                    form_key = agent_name + "__" + config.config_key
                    value_in_form = request.form.getlist(form_key)
                    cfg_v, _ = check_single_config(value_in_form, config)
                    current_conf.set_agent_conf(agent_name, config.config_key, cfg_v)
        current_conf.write_disk()
        # reflect new configuration
        return {"success": True}


def web_interface():
    app.run(debug=True, use_reloader=False, host='0.0.0.0')


def run_web_interface():
    t = Thread(target=web_interface, daemon=True)
    t.start()
