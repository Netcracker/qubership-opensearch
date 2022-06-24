import jsonpath
from robot.api import logger

class JsonpathLibrary(object):

    def get_items_by_path(self, json_dict, json_path):
        logger.info(f"Json: {json_dict}, path: {json_path}")
        match_object = jsonpath.jsonpath(json_dict, json_path)
        return match_object[0]

