import logging
import os
import re
import string
import sys

import requests

from get_tags import get_tag


HEADERS = {
    "Host": "hub.docker.com",
    "Accept": "application/json",
    "Accept-Language": "en-US;q=0.7,en;q=0.3",
    "Accept-Encoding": "gzip, deflate, br",
    "Referer": "https://hub.docker.com/search/?type=image",
    "Search-Version": "v3",
    "Connection": "keep-alive",
}

CHARS = string.ascii_letters + string.digits
MAX_EPOCHS = len(CHARS)**2


def get_search_url(search, page):
    """
    Create the search URL according to search params and page number.
    """
    url = (f"https://hub.docker.com/api/content/v1/products/" +
           f"search?architecture=arm,arm64,386,amd64&operating_system=linux" +
           f"&page={page}&page_size=100&q={search}&type=image")
    return url


def convert_to_base(num, base):
    """
    Convert a base-10 integer to a different base.
    """
    q = num//base
    r = num % base
    if (q == 0):
        return [r]
    else:
        return convert_to_base(q, base) + [r]


def build_search(epoch):
    """
    Build a search param given an epoch integer. Goes recursively deeper into \
    the search space, e.g.

    build_search(0) -> "A"
    build_search(326) -> "fv"
    build_search(1005326) -> "eAkU"
    """
    chars = [CHARS[i] for i in convert_to_base(epoch, len(CHARS))]
    return "".join(chars)


def write_images(images):
    """
    Write an iterable to stdout.
    """
    for item in sorted(images):
        print(item)


def main():
    log = logging.getLogger("list_docker_images")
    logging.basicConfig(level=logging.INFO)
    epoch = len(CHARS)
    images = set()
    previous_count = -1
    while(epoch < MAX_EPOCHS and previous_count != len(images)):

        previous_count = len(images)
        search = build_search(epoch)
        page = 1

        # Docker API fails if fetching result > 10k
        # So limit to page 100 with 100 results per page
        while page < 100:
            try:
                log.info(f"images:{len(images)} search:{search} page:{page} ")

                url = get_search_url(search, page)
                response = requests.get(url, headers=HEADERS)
                result = response.json()
                images = images | set([summary["slug"] for summary in result["summaries"]])

                # go out of the loop if we reach end
                # of results before 10k
                if result["next"] == "":
                    break
            # If program stops before end, write images in a file
            except (KeyboardInterrupt, SystemExit):
                write_images(images)
                sys.exit()
            except Exception:
                log.exception("failed when fetching %s", url)
            page += 1
        epoch += 1

    write_images(images)
    images_tags = set()
    for image in images:
        try:
            tag = get_tag(image)
            images_tags.add(":".join(image, tag))
            log.info("processed %s:%s", image, tag)
        except (KeyboardInterrupt, SystemExit):
            write_images(images_tags)
            sys.exit()
        except Exception:
            log.exception("failed when fetching tag for %s", image)
    write_images(images_tags)


if __name__ == "__main__":
    main()
