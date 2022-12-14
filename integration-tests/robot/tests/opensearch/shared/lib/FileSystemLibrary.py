import os


class FileSystemLibrary(object):

    @staticmethod
    def file_exists(file_path):
        return os.path.exists(file_path)
