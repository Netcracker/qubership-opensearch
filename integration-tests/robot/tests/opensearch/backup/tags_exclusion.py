def check_that_parameters_are_presented(environ, *variable_names) -> bool:
    for variable in variable_names:
        if not environ.get(variable):
            return False
    return True


def get_excluded_tags(environ) -> list:
    excluded_tags = []
    if not check_that_parameters_are_presented(environ,
                                               'OPENSEARCH_CURATOR_HOST',
                                               'OPENSEARCH_CURATOR_PORT'):
        excluded_tags.append('backup')
    if not check_that_parameters_are_presented(environ,
                                               'OPENSEARCH_CURATOR_USERNAME',
                                               'OPENSEARCH_CURATOR_PASSWORD'):
        excluded_tags.append('unauthorized_access')
    if "full_backup" not in environ.get('TAGS'):
        excluded_tags.append('full_backup')
    return excluded_tags
