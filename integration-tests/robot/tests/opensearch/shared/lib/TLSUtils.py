import os


class TLSUtils(object):

    @staticmethod
    def file_exists(file_path):
        return os.path.exists(file_path)
