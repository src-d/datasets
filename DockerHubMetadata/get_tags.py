import re

import requests

parser = re.compile('Bearer +realm="(.*)",service="(.*)",scope="(.*)"')

def get_manifest_url(image, tag):
    return (f"https://registry-1.docker.io/v2/{image}/manifests/"
            f"{tag}")

def get_tag_url(image):
    return f"https://registry-1.docker.io/v2/{image}/tags/list/?n=1"

def get_tag(image):
    # Official images case
    if "/" not in image:
        image = "library/" + image
    tag = "latest"

    # Getting the correct auth scope for the current manifest request
    manifest_url = get_manifest_url(image, tag)
    response = requests.get(manifest_url)
    header = response.headers["Www-Authenticate"]
    match = parser.match(header)
    realm, service, scope = match.groups()

    # Retrieve authentication token
    response = requests.get(f"https://auth.docker.io/token?realm={realm}"
                            f"&service={service}"
                            f"&scope={scope}")
    data = response.json()
    token = data["token"]
    # Try getting manifest for "latest" tag
    response = requests.get(manifest_url,
                            headers=dict(Authorization=f"Bearer {token}"))
    data = response.json()

    # When "latest" tag do not exist, the tag list must be fetched.
    if "errors" in data and data["errors"][0]["code"] == "MANIFEST_UNKNOWN":
        tags_url = get_tag_url(image)
        response = requests.get(tags_url, headers=dict(
            Authorization=f"Bearer {token}"))
        tags = response.json()
        tags = tags["tags"]
        return tags[0]
    else:
        return "latest"
