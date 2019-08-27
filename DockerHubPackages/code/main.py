

# retrieves node libs, python libs and disk usage information for a dockerhub image
import sys
import json
import os
import logging

from analyzer.utils import mount_docker_image, docker_cleanup

from analyzer.analyzers.base import get_base_info
from analyzer.analyzers.native_packages import get_native_packages_info
from analyzer.analyzers.python_packages import get_python_packages_info
from analyzer.analyzers.node_packages import get_node_packages_info

logger = logging.getLogger(__name__)

def get_image_info(image):
    """Retrieve information from a given dockerhub image.

    :param image: name of the docker image to get info from.
    :type image: string

    :return: Dictionnary containing image name, size, distro, version and
            package information. 
    :rtype: dict
    """
    with mount_docker_image(image) as image_path:

        distro, version, image_size = get_base_info(image_path)

        natives_packages_info = get_native_packages_info(image_path, distro)

        python_packages_info = get_python_packages_info(image_path)
        
        node_packages_info = get_node_packages_info(image_path)
    
    return {
        'image': image,
        'size': image_size,
        'distribution': distro,
        'version': version,
        'packages': {
            'native': natives_packages_info,
            'python3': python_packages_info,
            'node': node_packages_info
        }
    }

if __name__ == "__main__":
    
    images_file = sys.argv[1]
    output_folder = sys.argv[2]
    pid = int(sys.argv[3])
    njob = int(sys.argv[4])
    
    with open(images_file, 'r') as images:

        image_count = int(get_ipython().getoutput(f"find {output_folder} -type f | wc -l").pop())

        for i in range(pid + image_count):
            images.readline().replace('\n', '')
        
        image = images.readline().replace('\n', '')
        count = 0

        while image:
            try:

                if image.endswith(":latest"):
                    filename = output_folder + '/' + \
                        image[:2] + '/' + image.split(":").pop(0) + '.json'
                else:
                    filename = output_folder + '/' + \
                        image[:2] + '/' + image + '.json'

                if not os.path.isfile(filename):
                    image_data = get_image_info(image)

                    
                    os.makedirs(os.path.dirname(filename), exist_ok=True)
                    
                    with open(filename, "w") as output_file:
                        json.dump(image_data, output_file, indent=4)
                    logger.info("processed", image)
                else:
                    logger.info("found existing", image)

            except Exception as e:
                logger.error("failed processing", image, e)

            for i in range(njob - 1):
                images.readline().replace('\n', '')
            
            image = images.readline().replace('\n', '')
            
            count += 1

            # Remove all unused images and volumes
            # So the script do not end up filling all available disk space
            if count % 300:
                docker_cleanup()
