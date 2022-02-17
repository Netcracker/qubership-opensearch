def check_that_parameters_are_presented(environ, *variable_names) -> bool:
    for variable in variable_names:
        if not environ.get(variable):
            return False
    return True


def get_excluded_tags(environ) -> list:
    if not check_that_parameters_are_presented(environ,
                                               'IDENTITY_PROVIDER_URL',
                                               'IDENTITY_PROVIDER_USERNAME',
                                               'IDENTITY_PROVIDER_PASSWORD',
                                               'IDENTITY_PROVIDER_REGISTRATION_TOKEN'):
        return ['oauth']
