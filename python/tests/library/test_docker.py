# Copyright 2026 Axis Communications AB.
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
"""Tests for the Docker library retry logic."""

import asyncio
import logging
import sys
from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
import pytest

from etos_api.library.docker import Docker

logging.basicConfig(level=logging.DEBUG, stream=sys.stdout)


def _make_connector_error():
    """Create a ClientConnectorError using a mock ConnectionKey."""
    connection_key = MagicMock()
    connection_key.host = "ghcr.io"
    connection_key.port = 443
    connection_key.is_ssl = True
    connection_key.ssl = True
    return aiohttp.ClientConnectorError(connection_key, OSError("DNS resolution failed"))


class TestDockerDigestRetry:
    """Test retry logic for Docker.digest."""

    logger = logging.getLogger(__name__)
    pytestmark = pytest.mark.asyncio

    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_returns_on_first_success(self, mock_get_digest):
        """Test that digest returns immediately when first attempt succeeds.

        Approval criteria:
            - Digest shall be returned after a single successful attempt.

        Test steps::
            1. Call digest with a working image name.
            2. Verify _get_digest was called once and the digest is returned.
        """
        mock_get_digest.return_value = "sha256:abc123"
        docker = Docker()
        result = await docker.digest("ghcr.io/eiffel-community/etos-test-runner:latest")
        assert result == "sha256:abc123"
        assert mock_get_digest.call_count == 1

    @patch("etos_api.library.docker.BACKOFF_FACTOR", 0)
    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_retries_on_connector_error(self, mock_get_digest):
        """Test that digest retries on ClientConnectorError.

        Approval criteria:
            - Digest shall retry on transient connection errors and succeed.

        Test steps::
            1. Configure _get_digest to fail once with ClientConnectorError then succeed.
            2. Call digest.
            3. Verify retry happened and correct digest is returned.
        """
        mock_get_digest.side_effect = [
            _make_connector_error(),
            "sha256:abc123",
        ]
        docker = Docker()
        result = await docker.digest("ghcr.io/eiffel-community/etos-test-runner:latest")
        assert result == "sha256:abc123"
        assert mock_get_digest.call_count == 2

    @patch("etos_api.library.docker.BACKOFF_FACTOR", 0)
    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_retries_on_server_disconnected(self, mock_get_digest):
        """Test that digest retries on ServerDisconnectedError.

        Approval criteria:
            - Digest shall retry when the server disconnects unexpectedly.

        Test steps::
            1. Configure _get_digest to fail once with ServerDisconnectedError then succeed.
            2. Call digest.
            3. Verify retry happened and correct digest is returned.
        """
        mock_get_digest.side_effect = [
            aiohttp.ServerDisconnectedError(),
            "sha256:abc123",
        ]
        docker = Docker()
        result = await docker.digest("ghcr.io/eiffel-community/etos-test-runner:latest")
        assert result == "sha256:abc123"
        assert mock_get_digest.call_count == 2

    @patch("etos_api.library.docker.BACKOFF_FACTOR", 0)
    @patch("etos_api.library.docker.MAX_RETRIES", 3)
    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_returns_none_after_all_retries_exhausted(self, mock_get_digest):
        """Test that digest returns None when all retries are exhausted.

        Approval criteria:
            - Digest shall return None if every attempt fails with a connection error.

        Test steps::
            1. Configure _get_digest to always raise ClientConnectorError.
            2. Call digest.
            3. Verify None is returned and all retries were attempted.
        """
        error = _make_connector_error()
        mock_get_digest.side_effect = [error, error, error]
        docker = Docker()
        result = await docker.digest("ghcr.io/eiffel-community/etos-test-runner:latest")
        assert result is None
        assert mock_get_digest.call_count == 3

    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_does_not_retry_on_non_retryable_error(self, mock_get_digest):
        """Test that digest does not retry on non-retryable exceptions.

        Approval criteria:
            - Non-retryable exceptions shall propagate immediately without retries.

        Test steps::
            1. Configure _get_digest to raise a RuntimeError.
            2. Call digest.
            3. Verify the exception propagates and no retry occurred.
        """
        mock_get_digest.side_effect = RuntimeError("unexpected")
        docker = Docker()
        with pytest.raises(RuntimeError, match="unexpected"):
            await docker.digest("ghcr.io/eiffel-community/etos-test-runner:latest")
        assert mock_get_digest.call_count == 1

    @patch("etos_api.library.docker.BACKOFF_FACTOR", 0)
    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_retries_on_timeout_error(self, mock_get_digest):
        """Test that digest retries on asyncio.TimeoutError.

        Approval criteria:
            - Digest shall retry on timeout errors and succeed if a subsequent attempt works.

        Test steps::
            1. Configure _get_digest to fail once with TimeoutError then succeed.
            2. Call digest.
            3. Verify retry happened and correct digest is returned.
        """
        mock_get_digest.side_effect = [
            asyncio.TimeoutError(),
            "sha256:abc123",
        ]
        docker = Docker()
        result = await docker.digest("ghcr.io/eiffel-community/etos-test-runner:latest")
        assert result == "sha256:abc123"
        assert mock_get_digest.call_count == 2

    @patch("etos_api.library.docker.BACKOFF_FACTOR", 1)
    @patch("etos_api.library.docker.MAX_RETRIES", 4)
    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_uses_exponential_backoff(self, mock_get_digest):
        """Test that retry delays follow exponential backoff.

        Approval criteria:
            - Delays between retries shall follow BACKOFF_FACTOR * 2^(attempt-1).

        Test steps::
            1. Configure _get_digest to always fail with ServerDisconnectedError.
            2. Patch asyncio.sleep to record the delay values.
            3. Call digest.
            4. Verify the recorded delays match 1, 2, 4 (for attempts 1, 2, 3).
        """
        mock_get_digest.side_effect = aiohttp.ServerDisconnectedError()
        recorded_delays = []

        async def fake_sleep(delay):
            recorded_delays.append(delay)

        docker = Docker()
        with patch("etos_api.library.docker.asyncio.sleep", side_effect=fake_sleep):
            result = await docker.digest("ghcr.io/eiffel-community/etos-test-runner:latest")

        assert result is None
        assert mock_get_digest.call_count == 4
        # Delays: BACKOFF_FACTOR * 2^0, BACKOFF_FACTOR * 2^1, BACKOFF_FACTOR * 2^2
        # No sleep after the last attempt.
        assert recorded_delays == [1, 2, 4]


class TestDockerReferenceParsing:
    """Test parsing of image references (tag and digest) by Docker."""

    @pytest.mark.parametrize(
        "name, expected_base, expected_reference",
        [
            # Tag-based references.
            ("etos-test-runner", "etos-test-runner", "latest"),
            ("etos-test-runner:1.2.3", "etos-test-runner", "1.2.3"),
            (
                "ghcr.io/eiffel-community/etos-test-runner:latest",
                "ghcr.io/eiffel-community/etos-test-runner",
                "latest",
            ),
            # Registry with port and no tag must not confuse the port for a tag.
            (
                "registry.example.com:5000/etos-test-runner",
                "registry.example.com:5000/etos-test-runner",
                "latest",
            ),
            (
                "registry.example.com:5000/etos-test-runner:1.2.3",
                "registry.example.com:5000/etos-test-runner",
                "1.2.3",
            ),
            # Digest-pinned references must use the digest verbatim.
            (
                "ghcr.io/eiffel-community/etos-test-runner@sha256:" + "a" * 64,
                "ghcr.io/eiffel-community/etos-test-runner",
                "sha256:" + "a" * 64,
            ),
            (
                "etos-test-runner@sha256:" + "b" * 64,
                "etos-test-runner",
                "sha256:" + "b" * 64,
            ),
            # Registry with port and digest.
            (
                "registry.example.com:5000/etos-test-runner@sha256:" + "c" * 64,
                "registry.example.com:5000/etos-test-runner",
                "sha256:" + "c" * 64,
            ),
            # Both tag and digest present: the digest wins and the tag is dropped.
            (
                "ghcr.io/eiffel-community/etos-test-runner:1.2.3@sha256:" + "d" * 64,
                "ghcr.io/eiffel-community/etos-test-runner",
                "sha256:" + "d" * 64,
            ),
        ],
    )
    def test_reference(self, name, expected_base, expected_reference):
        """Test that reference returns the correct base image and reference.

        Approval criteria:
            - Tag-based references shall be parsed into base image and tag.
            - Digest-pinned references shall be parsed into base image and digest.

        Test steps::
            1. Call reference with an image name.
            2. Verify the returned base image and reference are correct.
        """
        docker = Docker()
        base, reference = docker.reference(name)
        assert base == expected_base
        assert reference == expected_reference


class TestDockerManifestUrl:
    """Test the manifest URL built by Docker.digest."""

    logger = logging.getLogger(__name__)
    pytestmark = pytest.mark.asyncio

    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_digest_pinned_image_builds_correct_manifest_url(self, mock_get_digest):
        """Test that a digest-pinned image produces a digest-based manifest URL.

        Approval criteria:
            - The manifest URL for a digest-pinned image shall reference the digest
              verbatim, not parse it as a tag.

        Test steps::
            1. Call digest with a digest-pinned image name.
            2. Verify the manifest URL passed to _get_digest uses the digest.
        """
        mock_get_digest.return_value = "sha256:" + "a" * 64
        docker = Docker()
        digest = "sha256:" + "a" * 64
        await docker.digest(f"ghcr.io/eiffel-community/etos-test-runner@{digest}")
        mock_get_digest.assert_awaited_once_with(
            f"https://ghcr.io/v2/eiffel-community/etos-test-runner/manifests/{digest}"
        )

    @patch.object(Docker, "_get_digest", new_callable=AsyncMock)
    async def test_tag_based_image_builds_correct_manifest_url(self, mock_get_digest):
        """Test that a tag-based image produces a tag-based manifest URL.

        Approval criteria:
            - The manifest URL for a tag-based image shall reference the tag.

        Test steps::
            1. Call digest with a tag-based image name.
            2. Verify the manifest URL passed to _get_digest uses the tag.
        """
        mock_get_digest.return_value = "sha256:abc123"
        docker = Docker()
        await docker.digest("ghcr.io/eiffel-community/etos-test-runner:1.2.3")
        mock_get_digest.assert_awaited_once_with(
            "https://ghcr.io/v2/eiffel-community/etos-test-runner/manifests/1.2.3"
        )
