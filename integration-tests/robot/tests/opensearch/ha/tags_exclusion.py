def get_excluded_tags(environ) -> list:
    if environ.get("EXTERNAL_OPENSEARCH", False):
        return ['ha']
