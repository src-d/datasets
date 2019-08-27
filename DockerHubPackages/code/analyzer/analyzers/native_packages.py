from ..utils import run
import logging

logger = logging.getLogger(__name__)

def get_package_info(package, distro, path):
    """Get details about one precise native package in the given image for the given distro.

    :param distro: expected distribution.
    :type distro: string
    :param package: name of the python package to get info from.
    :type package: string
    :param path: path were the docker image filesystem is expanded.
    :type path: string

    :return: list containing package name, version and size
    :rtype: list[string, string, int]
    """
    if distro in ['debian']:
        name, version = package.split(' ')
        command = f"sudo chroot {path} dpkg -L {name} | xargs sudo chroot {path} stat -c '%s' | paste -s -d+ | bc 2>/dev/null"
        size = get_ipython().getoutput(command).pop()
        return [name, version, int(size)]

    if distro in ['ubuntu']:
        name, version, size = package.split('\t')[:3]
        return [name, version, int(size)]

    if distro in ['arch']:
        name, version = package.split(' ')
        command = f"sudo chroot {path} pacman -Qi {name} 2>/dev/null"
        info = get_ipython().getoutput(command)
        powers = {"GiB": 1e9, "MiB": 1e6, "KiB": 1e3, "B": 1}
        for line in info:
            splitted_line = line.split(" ")
            if "Size" in splitted_line:
                power = splitted_line.pop()
                size = float(splitted_line.pop())*powers[power]
        return [name, version, int(size)]

    if distro in ['alpine']:
        command = f"sudo chroot {path} apk info --size {package} 2>/dev/null"
        size = get_ipython().getoutput(command).pop(1)
        name, version = package.split(' ')
        return [name, version, int(size)]

    if distro in ['centos', 'fedora', 'ol', 'amzn']:
        command = f"sudo chroot {path} rpm -qi {package} 2>/dev/null"
        info = get_ipython().getoutput(command)
        for line in info:
            splitted_line = line.split(" ")
            if "Name" in splitted_line:
                name = splitted_line.pop()
            if "Version" in splitted_line:
                version = splitted_line.pop()
            if "Release" in splitted_line:
                release = splitted_line.pop()
            if "Size" in splitted_line:
                size = splitted_line.pop()
        return [name, f'{version}.{release}', int(size)]


def get_native_packages_list(distro, path):
    """List all native packages in an image filesystem given it's distro.

    :param distro: expected distribution name.
    :type distro: string
    :param path: path were the docker image filesystem is expanded.
    :type path: string

    :return: list of each package's name
    :rtype: list[string]
    """
    if distro in ['debian']:
        command = f"sudo chroot {path} dpkg -l 2>/dev/null | sed -e '1,/+++/d' | tr -s ' ' | cut -d ' ' -f 2,3 2>/dev/null"

    elif distro in ['ubuntu']:
        command = f"sudo chroot {path}" + " dpkg-query -Wf '${Package}\t${Version}\t${Installed-Size}\n' 2>/dev/null"

    elif distro in ['arch']:
        command = f"sudo chroot {path} pacman -Q 2>/dev/null"

    elif distro in ['alpine']:
        command = f"sudo chroot {path} apk info -v 2>/dev/null | rev | sed 's/-/ /2'  | rev 2>/dev/null"

    elif distro in ['centos', 'fedora', 'ol', 'amzn']:
        command = f"sudo chroot {path} rpm -qa"
    else:
        return []

    natives_packages = get_ipython().getoutput(command)

    return natives_packages

def get_native_packages_info(path, distro):
    """Get details about all native packages in an image filesystem given expected distro.
    
    :param path: path were the docker image filesystem is expanded.
    :type path: string
    :param distro: expected distribution name.
    :type distro: string

    :return: list containing lists of each package's name, version and size
    :rtype: list[list[string, string, int]]
    """
    native_packages = get_native_packages_list(distro, path)
    native_packages_info = []

    for i, package in enumerate(native_packages):
        try:
            native_packages_info.append(
                get_package_info(package, distro, path))
        except Exception as e:
            logger.error("Exception", e,
                         "when prorcessing", package, distro, path)

    return native_packages_info
