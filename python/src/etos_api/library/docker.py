# Copyright 2023 Axis Communications AB.
#
# For a full list of individual contributors, please see the commit history.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Docker operations for the ETOS API."""
import time
import logging
from typing import Optional, Mapping
from threading import Lock
import aiohttp

DEFAULT_TAG = "latest"
DEFAULT_REGISTRY = "index.docker.io"
REPO_DELIMITER = "/"
TAG_DELIMITER = ":"


class Docker:
    """Docker handler for HTTP operations against docker registries.

    This handler is heavily inspired by `crane digest`:
    https://github.com/google/go-containerregistry/tree/main/cmd/crane
    """

    logger = logging.getLogger(__name__)
    # The amount of time in seconds before the token expires where we should recreate it.
    token_expire_modifier = 30
    # In-memory database for stored authorization tokens.
    # This dictionary shares memory with all instances of `Docker`, by design.
    tokens = {}
    lock = Lock()

    def token(self, manifest_url: str) -> Optional[str]:
        """Get a stored token, removing it if expired.

        :param manifest_url: URL the token has been stored for.
        :return: A token or None.
        """
        with self.lock:
            token = self.tokens.get(manifest_url, {})
            if token:
                if time.time() >= token["expire"]:
                    self.logger.info("Registry token expired for %r", manifest_url)
                    self.tokens.pop(manifest_url)
                    token = {}
            return token.get("token")

    async def head(
        self, session: aiohttp.ClientSession, url: str, token: Optional[str] = None
    ) -> aiohttp.ClientResponse:
        """Make a HEAD request to a URL, adding token to headers if supplied.

        :param session: Client HTTP session to use for HTTP request.
        :param url: URL to make HEAD request to.
        :param token: Optional authorization token.
        :return: HTTP response.
        """
        headers = {}
        if token is not None:
            headers["Authorization"] = f"Bearer {token}"

        async with session.head(url, headers=headers) as response:
            return response

    async def get_token_from_container_registry(
        self, session: aiohttp.ClientSession, realm: str, parameters: dict
    ) -> dict:
        """Get a token from an unauthorized request to container repository.

        :param session: Client HTTP session to use for HTTP request.
        :param realm: The realm to authorize against.
        :param parameters: Parameters to use for the authorization request.
        :return: Response JSON from authorization request.
        """
        async with session.get(realm, params=parameters) as response:
            response.raise_for_status()
            return await response.json()

    async def authorize(
        self, session: aiohttp.ClientSession, response: aiohttp.ClientResponse, manifest_url: str
    ) -> str:
        """Authorize against container registry.

        :param session: Client HTTP session to use for HTTP request.
        :param response: Response from a previous request to parse headers from.
        :param manifest_url: Manifest URL to query should a new auth request be needed.
        :return: A token retrieved from container registry.
        """
        parameters = await self.parse_headers(response.headers)
        url = parameters.get("realm")
        if not isinstance(url, str) or not (
            url.startswith("http://") or url.startswith("https://")
        ):
            self.logger.warning("No realm in original request, retrying without a token")
            with self.lock:
                try:
                    del self.tokens[manifest_url]
                except KeyError:
                    pass
            response = await self.head(session, manifest_url)
            parameters = await self.parse_headers(response.headers)
            url = parameters.get("realm")
            if not isinstance(url, str) or not (
                url.startswith("http://") or url.startswith("https://")
            ):
                raise ValueError(
                    f"No realm URL found in www-authenticate header: {response.headers}"
                )
        url = parameters.pop("realm")
        response_json = await self.get_token_from_container_registry(session, url, parameters)
        with self.lock:
            self.tokens[manifest_url] = {
                "token": response_json.get("token"),
                "expire": time.time()
                + response_json.get("expires_in", 0.0)
                - self.token_expire_modifier,
            }
        return ""

    async def parse_headers(self, headers: Mapping) -> dict:
        """Parse the www-authenticate header and convert it to a dict.

        :param headers: Headers to parse.
        :return: Dictionary of the keys in the www-authenticate header.
        """
        www_auth_header = headers.get("www-authenticate")
        if www_auth_header is None:
            return {}
        challenge = www_auth_header.replace("Bearer ", "")
        parts = challenge.split(",")

        parameters = {}
        for part in parts:
            key, value = part.split("=")
            parameters[key] = value.strip('"')
        return parameters

    def tag(self, base: str) -> tuple[str, str]:
        """Figure out tag from a container image name.

        :param base: Name of image.
        :return: Base image name without tag and tag name.
        """
        tag = ""

        parts = base.split(TAG_DELIMITER)
        # Verify that we aren't confusing a tag for a hostname w/ port
        # If there are more than one ':' in the image name, we'll assume
        # that the container tag is after the second ':', not the first.
        # By checking if the first part (i.e. hostname w/ port) does not
        # have any '/', we will also catch cases where there's a
        # hostname w/ port and no tag in the image name.
        if len(parts) > 1 and REPO_DELIMITER not in parts[-1]:
            base = TAG_DELIMITER.join(parts[:-1])
            tag = parts[-1]
            self.logger.info("Assuming tag is %r", tag)
        if tag == "":
            tag = DEFAULT_TAG
            self.logger.info("Assuming default tag %r", tag)
        return base, tag

    def repository(self, repo: str) -> tuple[str, str]:
        """Figure out repository and registry from a container image name.

        :param repo: Name of image, including or excluding registry URL.
        :return: Registry URL and the repo path for that registry.
        """
        registry = ""
        parts = repo.split(REPO_DELIMITER, 1)
        if len(parts) == 2 and ("." in parts[0] or ":" in parts[0]):
            # The first part of the repository is treated as the registry domain
            # if it contains a '.' or ':' character, otherwise it is all repository
            # and the domain defaults to Docker Hub.
            registry = parts[0]
            repo = parts[1]
            self.logger.info(
                "Probably found a registry URL in image name: (registry=%r, repo=%r)",
                registry,
                repo,
            )
        if registry in ("", "docker.io"):
            self.logger.info("Assuming registry is %r for %r", DEFAULT_REGISTRY, repo)
            registry = DEFAULT_REGISTRY
        return registry, repo

    async def digest(self, name: str) -> Optional[str]:
        """Get a sha256 digest from an image in an image repository.

        :param name: The name of the container image.
        :return: The sha256 digest of the container image.
        """
        self.logger.info("Figure out digest for %r", name)
        base, tag = self.tag(name)
        registry, repo = self.repository(base)
        manifest_url = f"https://{registry}/v2/{repo}/manifests/{tag}"

        digest = None
        async with aiohttp.ClientSession() as session:
            self.logger.info("Get digest from %r", manifest_url)
            response = await self.head(session, manifest_url, self.token(manifest_url))
            try:
                if response.status == 401 and "www-authenticate" in response.headers:
                    self.logger.info("Generate a new authorization token for %r", manifest_url)
                    token = await self.authorize(session, response, manifest_url)
                    response = await self.head(session, manifest_url, token)
                digest = response.headers.get("Docker-Content-Digest")
            except aiohttp.ClientResponseError as exception:
                self.logger.error("Error getting container image %r", exception)
                digest = None
            except ValueError as exception:
                self.logger.error("Failed to authenticate with container registry: %r", exception)
                digest = None
        self.logger.info("Returning digest %r from %r", digest, manifest_url)
        return digest
