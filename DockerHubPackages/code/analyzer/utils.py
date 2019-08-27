import subprocess

def run(command):
    """Run a bash command from python and returns the output, splitted line by line.
    
    :param command: command to run.
    :type command: string

    :return: list of the stdout lines.
    :rtype: list[string]
    """
    result = subprocess.run([command], stdout=subprocess.PIPE, shell=True)
    return result.stdout.decode("utf-8").split("\n")[:-1]

def docker_cleanup():
    """Remove all unused docker images and volumes stored on disk."""
    get_ipython().getoutput("docker system prune -af && docker volume prune -f")

class mount_docker_image:
    """High level wrapper around a docker image filesystem."""

    def __init__(self, image):
        """Expand the filesystem of the given docker image and stores its path."""
        # Running the container
        container = get_ipython().getoutput(f'sudo docker create {image}').pop()

        # Copying its content in a temporary directory
        tmp_dir = get_ipython().getoutput(f'mktemp -d -t dc-XXXXXXXXXX').pop()
        tmp_file = get_ipython().getoutput(f'mktemp -t fc-XXXXXXXXXX').pop()
        get_ipython().getoutput(f'docker export  {container} > {tmp_file}')
        get_ipython().getoutput(f'docker rm -f {container} > /dev/null')
        get_ipython().getoutput(f'tar fx {tmp_file} -C {tmp_dir}')
        get_ipython().getoutput(f'sudo rm -rf {tmp_file}')
        
        self.dir = tmp_dir


    def __enter__(self):
        """Return the directory where the image filesystem is expanded."""
        return self.dir

    def __exit__(self, type, value, traceback):
        """Clean the image filesystem."""
        get_ipython().getoutput(f'sudo rm -rf {self.dir}')

