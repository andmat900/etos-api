# Copyright 2020-2021 Axis Communications AB.
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
"""Environment provider proxy API."""
import logging
import asyncio
import os
import time
import aiohttp
from fastapi import APIRouter, HTTPException
from etos_lib import ETOS

from .schemas import ConfigureEnvironmentProviderRequest

ROUTER = APIRouter()
LOGGER = logging.getLogger(__name__)


async def _wait_for_configuration(etos_library, environment):
    """Wait for the environment provider configuration to apply.

    :param etos_library: An ETOS library instance for requesting the environment provider.
    :type etos_library: :obj:`etos_lib.ETOS`
    :param environment: Environment that has been configured.
    :type environment: :obj:`etos_api.routers.etos.schemas.ConfigureEnvironmentProviderRequest`
    """
    LOGGER.info("Waiting for configuration to be applied in the environment provider.")
    end_time = time.time() + etos_library.debug.default_http_timeout
    LOGGER.debug("Timeout: %r", etos_library.debug.default_http_timeout)
    async with aiohttp.ClientSession() as session:
        while time.time() < end_time:
            try:
                async with session.get(
                    f"{etos_library.debug.environment_provider}/configure",
                    params={"suite_id": environment.suite_id},
                    headers={
                        "Content-Type": "application/json",
                        "Accept": "application/json",
                    },
                ) as response:
                    assert 200 <= response.status < 400
                    response_json = await response.json()
                    LOGGER.info("Configuration: %r", response_json)
                    assert response_json.get("dataset") is not None
                    assert response_json.get("iut_provider") is not None
                    assert response_json.get("log_area_provider") is not None
                    assert response_json.get("execution_space_provider") is not None
                break
            except AssertionError:
                if response.status < 400:
                    LOGGER.warning("Configuration not ready yet.")
                else:
                    LOGGER.warning(
                        "Configuration verification request failed: %r, %r",
                        response.status,
                        response.reason,
                    )
                await asyncio.sleep(2)
        else:
            raise HTTPException(
                status_code=400,
                detail="Environment provider configuration did not apply properly",
            )


@ROUTER.post(
    "/environment_provider/configure", tags=["environment_provider"], status_code=204
)
async def configure_environment_provider(
    environment: ConfigureEnvironmentProviderRequest,
):
    """Configure environment provider request.

    :param environment: Environment to configure.
    :type environment: :obj:`etos_api.routers.etos.schemas.ConfigureEnvironmentProviderRequest`
    """
    LOGGER.identifier.set(environment.suite_id)
    LOGGER.info("Configuring environment provider using %r", environment)
    etos_library = ETOS("ETOS API", os.getenv("HOSTNAME"), "ETOS API")

    end_time = time.time() + etos_library.debug.default_http_timeout
    LOGGER.debug("HTTP Timeout: %r", etos_library.debug.default_http_timeout)
    async with aiohttp.ClientSession() as session:
        while time.time() < end_time:
            try:
                async with session.post(
                    f"{etos_library.debug.environment_provider}/configure",
                    json=environment.dict(),
                    headers={
                        "Content-Type": "application/json",
                        "Accept": "application/json",
                    },
                ) as response:
                    assert 200 <= response.status < 400
                break
            except AssertionError:
                LOGGER.warning(
                    "Configuration request failed: %r, %r",
                    response.status,
                    response.reason,
                )
                await asyncio.sleep(2)
        else:
            raise HTTPException(
                status_code=400,
                detail=f"Unable to configure environment provider with '{environment.json()}'",
            )
        await _wait_for_configuration(etos_library, environment)
