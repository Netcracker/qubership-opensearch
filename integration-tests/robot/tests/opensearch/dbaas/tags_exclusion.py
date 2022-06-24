def check_that_parameters_are_presented(environ, *variable_names) -> bool:
    for variable in variable_names:
        if not environ.get(variable):
            return False
    return True


def get_excluded_tags(environ) -> list:
    if not check_that_parameters_are_presented(environ,
                                               'OPENSEARCH_DBAAS_ADAPTER_HOST',
                                               'OPENSEARCH_DBAAS_ADAPTER_PORT',
                                               'OPENSEARCH_DBAAS_ADAPTER_USERNAME',
                                               'OPENSEARCH_DBAAS_ADAPTER_PASSWORD',
                                               'OPENSEARCH_DBAAS_ADAPTER_REPOSITORY'):
        return ['dbaas']
    if environ.get("OPENSEARCH_DBAAS_ADAPTER_API_VERSION") != "v2":
        return ['dbaas_v2']
    else:
        return ['dbaas_v1']
