# Copyright 2020 Axis Communications AB.
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
"""ETOS API routers."""
import logging
import sys
from unittest.mock import patch, AsyncMock
from fastapi.testclient import TestClient
from etos_lib.lib.debug import Debug
from etos_api.main import APP

logging.basicConfig(level=logging.DEBUG, stream=sys.stdout)


class TestRouters:
    """Test the routers in etos-api."""

    logger = logging.getLogger(__name__)
    client = TestClient(APP)

    @staticmethod
    def teardown_method():
        """Cleanup events from ETOS library debug."""
        Debug().events_received.clear()
        Debug().events_published.clear()

    def test_head_on_root_without_redirect(self):
        """Test that HEAD requests on root return 308 permanent redirect.

        Approval criteria:
            - A HEAD request on root shall return 308.

        Test steps::
            1. Send a HEAD request to root without allow_redirects.
            2. Verify that status code is 308.
        """
        self.logger.info("STEP: Send a HEAD request to root without allow_redirects.")
        response = self.client.head("/", allow_redirects=False)
        self.logger.info("STEP: Verify that status code is 308.")
        assert response.status_code == 308

    def test_head_on_root_with_redirect(self):
        """Test that HEAD requests on root return 204 when redirected.

        Approval criteria:
            - A redirected HEAD request on root shall return 204.

        Test steps::
            1. Send a HEAD request to root with allow_redirects.
            2. Verify that status code is 204.
        """
        self.logger.info("STEP: Send a HEAD request to root with allow_redirects.")
        response = self.client.head("/", allow_redirects=True)
        self.logger.info("STEP: Verify that status code is 204.")
        assert response.status_code == 204

    def test_post_on_root_without_redirect(self):
        """Test that POST requests on root return 308 permanent redirect.

        Approval criteria:
            - A POST request on root shall return 308.

        Test steps::
            1. Send a POST request to root without allow_redirects.
            2. Verify that status code is 308.
        """
        self.logger.info("STEP: Send a POST request to root without allow_redirects.")
        response = self.client.post("/", allow_redirects=False)
        self.logger.info("STEP: Verify that status code is 308.")
        assert response.status_code == 308

    @patch("etos_api.library.validator.Docker.digest")
    @patch("etos_api.library.validator.SuiteValidator._download_suite")
    @patch("etos_api.library.graphql.GraphqlQueryHandler.execute")
    @patch("etos_api.routers.environment_provider.router.aiohttp.ClientSession")
    def test_post_on_root_with_redirect(
        self, mock_client, graphql_execute_mock, download_suite_mock, digest_mock
    ):
        """Test that POST requests to / redirects and starts ETOS tests.

        Approval criteria:
            - A redirected POST requests to root shall return 200.

        Test steps::
            1. Send a POST request to root with allow_redirects.
            2. Verify that the status code is 200.
        """
        digest_mock.return_value = (
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        )

        mock_client().__aenter__.return_value = mock_client
        mock_client.post().__aenter__.return_value = mock_client
        mock_client.get().__aenter__.return_value = mock_client
        mock_client.json = AsyncMock(
            return_value={
                "dataset": "{}",
                "iut_provider": "iut",
                "log_area_provider": "log_area",
                "execution_space_provider": "execution_space",
            }
        )
        mock_client.status = 200
        # post is called above when adding the __aenter__ return_value.
        mock_client.post.reset_mock()

        graphql_execute_mock.return_value = {
            "artifactCreated": {
                "edges": [
                    {
                        "node": {
                            "meta": {"id": "cda58701-5614-49bf-9101-11b71a5721fb"},
                            "data": {"identity": "pkg:eiffel-community/etos-api"},
                        }
                    }
                ]
            }
        }
        download_suite_mock.return_value = [
            {
                "name": "TestRouters",
                "priority": 1,
                "recipes": [
                    {
                        "constraints": [
                            {"key": "ENVIRONMENT", "value": {}},
                            {"key": "PARAMETERS", "value": {}},
                            {"key": "COMMAND", "value": "exit 0"},
                            {"key": "TEST_RUNNER", "value": "TestRunner"},
                            {"key": "EXECUTE", "value": []},
                            {"key": "CHECKOUT", "value": ["echo 'checkout'"]},
                        ],
                        "id": "132a7499-7ad4-4c4a-8a66-4e9ac95c7885",
                        "testCase": {
                            "id": "test_start_etos",
                            "tracker": "Github",
                            "url": "https://github.com/eiffel-community/etos-api",
                        },
                    }
                ],
            }
        ]

        self.logger.info("STEP: Send a POST request to root with allow_redirects.")
        response = self.client.post(
            "/",
            json={
                "artifact_identity": "pkg:testing/etos",
                "test_suite_url": "http://localhost/my_test.json",
            },
            allow_redirects=True,
        )
        self.logger.info("STEP: Verify that the status code is 200.")
        assert response.status_code == 200

    @patch("etos_api.library.validator.Docker.digest")
    @patch("etos_api.library.validator.SuiteValidator._download_suite")
    @patch("etos_api.library.graphql.GraphqlQueryHandler.execute")
    @patch("etos_api.routers.environment_provider.router.aiohttp.ClientSession")
    def test_start_etos(self, mock_client, graphql_execute_mock, download_suite_mock, digest_mock):
        """Test that POST requests to /etos attempts to start ETOS tests.

        Approval criteria:
            - POST requests to ETOS shall return 200.
            - POST requests to ETOS shall attempt to send TERCC.
            - POST requests to ETOS shall configure environment provider.

        Test steps::
            1. Send a POST request to etos.
            2. Verify that the status code is 200.
            3. Verify that a TERCC was sent.
            4. Verify that the environment provider was configured.
        """
        digest_mock.return_value = (
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        )
        mock_client().__aenter__.return_value = mock_client
        mock_client.post().__aenter__.return_value = mock_client
        mock_client.get().__aenter__.return_value = mock_client
        mock_client.json = AsyncMock(
            return_value={
                "dataset": "{}",
                "iut_provider": "iut",
                "log_area_provider": "log_area",
                "execution_space_provider": "execution_space",
            }
        )
        mock_client.status = 200
        # post is called above when adding the __aenter__ return_value.
        mock_client.post.reset_mock()

        graphql_execute_mock.return_value = {
            "artifactCreated": {
                "edges": [
                    {
                        "node": {
                            "meta": {"id": "cda58701-5614-49bf-9101-11b71a5721fb"},
                            "data": {"identity": "pkg:eiffel-community/etos-api"},
                        }
                    }
                ]
            }
        }
        download_suite_mock.return_value = [
            {
                "name": "TestRouters",
                "priority": 1,
                "recipes": [
                    {
                        "constraints": [
                            {"key": "ENVIRONMENT", "value": {}},
                            {"key": "PARAMETERS", "value": {}},
                            {"key": "COMMAND", "value": "exit 0"},
                            {"key": "TEST_RUNNER", "value": "TestRunner"},
                            {"key": "EXECUTE", "value": []},
                            {"key": "CHECKOUT", "value": ["echo 'checkout'"]},
                        ],
                        "id": "132a7499-7ad4-4c4a-8a66-4e9ac95c7885",
                        "testCase": {
                            "id": "test_start_etos",
                            "tracker": "Github",
                            "url": "https://github.com/eiffel-community/etos-api",
                        },
                    }
                ],
            }
        ]
        self.logger.info("STEP: Send a POST request to etos.")
        response = self.client.post(
            "/etos",
            json={
                "artifact_identity": "pkg:testing/etos",
                "test_suite_url": "http://localhost/my_test.json",
            },
        )
        self.logger.info("STEP: Verify that the status code is 200.")
        assert response.status_code == 200
        self.logger.info("STEP: Verify that a TERCC was sent.")
        debug = Debug()
        tercc = None
        for event in debug.events_published:
            if event.meta.type == "EiffelTestExecutionRecipeCollectionCreatedEvent":
                tercc = event
                break
        assert tercc is not None
        assert response.json().get("tercc") == tercc.meta.event_id
        self.logger.info("STEP: Verify that the environment provider was configured.")
        mock_client.post.assert_called_once_with(
            f"{debug.environment_provider}/configure",
            json={
                "suite_id": tercc.meta.event_id,
                "dataset": {},
                "execution_space_provider": "default",
                "iut_provider": "default",
                "log_area_provider": "default",
            },
            headers={"Content-Type": "application/json", "Accept": "application/json"},
        )

    @patch("etos_api.routers.environment_provider.router.aiohttp.ClientSession")
    def test_configure_environment_provider(self, mock_client):
        """Test that configure requests are proxied to the environment provider.

        Approval criteria:
            - Requests to configure shall be redirected to the environment provider.
            - HTTP status code shall be 204.

        Test steps::
            1. Send a POST request to configure.
            2. Verify that the status code is 204.
            3. Verify that the request was sent to the environment provider.
        """
        mock_client().__aenter__.return_value = mock_client
        mock_client.post().__aenter__.return_value = mock_client
        mock_client.get().__aenter__.return_value = mock_client
        mock_client.json = AsyncMock(
            return_value={
                "dataset": "{}",
                "iut_provider": "iut",
                "log_area_provider": "log_area",
                "execution_space_provider": "execution_space",
            }
        )
        mock_client.status = 200
        mock_client.post.reset_mock()

        self.logger.info("STEP: Send a POST request to configure.")
        response = self.client.post(
            "environment_provider/configure",
            json={
                "suite_id": "f5d5bc7b-c6b8-406f-a997-43c8217e32c1",
                "dataset": {},
                "iut_provider": "iut",
                "execution_space_provider": "execution_space",
                "log_area_provider": "log_area",
            },
        )

        self.logger.info("STEP: Verify that the status code is 204.")
        assert response.status_code == 204

        self.logger.info("STEP: Verify that the request was sent to the environment provider.")
        debug = Debug()
        mock_client.post.assert_called_once_with(
            f"{debug.environment_provider}/configure",
            json={
                "suite_id": "f5d5bc7b-c6b8-406f-a997-43c8217e32c1",
                "dataset": {},
                "execution_space_provider": "execution_space",
                "iut_provider": "iut",
                "log_area_provider": "log_area",
            },
            headers={"Content-Type": "application/json", "Accept": "application/json"},
        )

    def test_selftest_get_ping(self):
        """Test that selftest ping with HTTP GET pings the system.

        Approval criteria:
            - GET requests to selftest ping shall return status code 204.

        Test steps::
            1. Send a GET request to selftest ping.
            2. Verify that status code is 204.
        """
        self.logger.info("STEP: Send a GET request to selftest ping.")
        response = self.client.get("/selftest/ping")
        self.logger.info("STEP: Verify that the status code is 204.")
        assert response.status_code == 204

    def test_selftest_head_ping(self):
        """Test that selftest ping with HTTP HEAD pings the system.

        Approval criteria:
            - HEAD requests to selftest ping shall return status code 204.

        Test steps::
            1. Send a HEAD request to selftest ping.
            2. Verify that the status code is 204.
        """
        self.logger.info("STEP: Send a HEAD request to selftest ping.")
        response = self.client.head("/selftest/ping")
        self.logger.info("STEP: Verify that the status code is 204.")
        assert response.status_code == 204
