import os

from ..utils import run
import logging

NPM_PATCH_PATH = os.getenv("NPM_PATCH_PATH")

logger = logging.getLogger(__name__)

def process_node_packages(package_folders):
    """Get details about all nodejs packages in a list of top-level packages folders.

    :param package_folders: paths were are located all the top-level package.json's.
    :type path: list[string]

    :return: list containing lists of each package's name, version and size
    :rtype: list[list[string, string, int]]
    """
    package_list = []
    for folder in package_folders:
        result = get_ipython().getoutput(
            f'(cd {folder}; {NPM_PATCH_PATH + "/bin/npm-cli.js"} ls)')
        for line in result:
            try:
                name, version, path = line.split(' ').pop().split('@')
                size = get_ipython().getoutput(
                    f'du --max-depth=0 --exclude=./node_modules {path}').pop().split('\t').pop(0)
                package_list.append([name, version, int(size)])
            except Exception as e:
                logger.error(e)
                pass
    return package_list

def get_node_packages_info(path):
    """Get details about all nodejs packages in an image filesystem.

    :param path: path were the docker image filesystem is expanded.
    :type path: string

    :return: list containing lists of each package's name, version and size
    :rtype: list[list[string, string, int]]
    """
    node_modules = get_ipython().getoutput(
            f'find {path} -name node_modules -type d -not -path "*/node_modules/*"')
    return process_node_packages(node_modules)
    
