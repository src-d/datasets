import sys
import json
import logging

# Create a custom logger
from ..utils import run

logger = logging.getLogger(__name__)

def get_dir_size(directory):
    """Get the size (in kb) of a directory given its path."""

    size = get_ipython().getoutput(f'sudo du -s {directory}/ | cut -f1').pop()
    return int(size)

def get_base_info(path):
   """Get basic info (distro, version, size) of a docker image given the path to its filesytem."""
   distro, version = get_ipython().getoutput(
       f'bash extract_packages.sh {path}').pop(0).split(':')
   
   size = get_dir_size(path)
   logger.info("detected size ")
   
   return distro, version, size
