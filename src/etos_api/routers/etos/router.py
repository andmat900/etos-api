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
"""ETOS API router."""
import logging
from uuid import uuid4
import os
from fastapi import APIRouter, HTTPException
from starlette.responses import Response
from etos_lib import ETOS
from etos_lib.logging.logger import FORMAT_CONFIG
from eiffellib.events import EiffelTestExecutionRecipeCollectionCreatedEvent

from etos_api.library.validator import SuiteValidator
from etos_api.library.utilities import sync_to_async, aclosing
from etos_api.routers.environment_provider.router import configure_environment_provider
from etos_api.routers.environment_provider.schemas import (
    ConfigureEnvironmentProviderRequest,
)
from .schemas import StartEtosRequest, StartEtosResponse
from .utilities import wait_for_artifact_created

ROUTER = APIRouter()
LOGGER = logging.getLogger(__name__)
logging.getLogger("pika").setLevel(logging.WARNING)


@ROUTER.post("/etos", tags=["etos"], response_model=StartEtosResponse)
async def start_etos(etos: StartEtosRequest):
    """Start ETOS execution on post.

    :param etos: ETOS pydantic model.
    :type etos: :obj:`etos_api.routers.etos.schemas.StartEtosRequest`
    :return: JSON dictionary with response.
    :rtype: dict
    """
    tercc = EiffelTestExecutionRecipeCollectionCreatedEvent()
    LOGGER.identifier.set(tercc.meta.event_id)

    LOGGER.info("Validating test suite.")
    try:
        await SuiteValidator().validate(etos.test_suite_url)
    except AssertionError as exception:
        LOGGER.error("Test suite validation failed!")
        LOGGER.error(exception)
        raise HTTPException(
            status_code=400, detail=f"Test suite validation failed. {exception}"
        ) from exception
    LOGGER.info("Test suite validated.")

    etos_library = ETOS("ETOS API", os.getenv("HOSTNAME"), "ETOS API")
    await sync_to_async(etos_library.config.rabbitmq_publisher_from_environment)

    LOGGER.info(
        "Get artifact created %r", (etos.artifact_identity or str(etos.artifact_id))
    )
    try:
        artifact = await wait_for_artifact_created(
            etos_library, etos.artifact_identity, etos.artifact_id
        )
    except Exception as exception:  # pylint:disable=broad-except
        LOGGER.critical(exception)
        raise HTTPException(
            status_code=400, detail=f"Could not connect to GraphQL. {exception}"
        ) from exception
    if artifact is None:
        raise HTTPException(
            status_code=400,
            detail=f"Unable to find artifact with identity '{etos.artifact_identity or str(etos.artifact_id)}'",
        )
    LOGGER.info("Found artifact created %r", artifact)
    # There are assumptions here. Since "edges" list is already tested
    # and we know that the return from GraphQL must be 'node'.'meta'.'id'
    # if there are "edges", this is fine.
    # Same goes for 'data'.'identity'.
    artifact_id = artifact[0]["node"]["meta"]["id"]
    identity = artifact[0]["node"]["data"]["identity"]

    links = {"CAUSE": artifact_id}
    data = {
        "selectionStrategy": {"tracker": "Suite Builder", "id": str(uuid4())},
        "batchesUri": etos.test_suite_url,
    }
    request = ConfigureEnvironmentProviderRequest(
        suite_id=tercc.meta.event_id,
        dataset=etos.dataset,
        execution_space_provider=etos.execution_space_provider,
        iut_provider=etos.iut_provider,
        log_area_provider=etos.log_area_provider,
    )
    try:
        await configure_environment_provider(request)
    except Exception as exception:  # pylint:disable=broad-except
        LOGGER.critical(exception)
        raise HTTPException(
            status_code=400,
            detail=f"Could not configure environment provider. {exception}",
        ) from exception
    LOGGER.info("Environment provider configured.")

    LOGGER.info("Start event publisher.")
    await sync_to_async(etos_library.start_publisher)
    await sync_to_async(etos_library.publisher.wait_start)
    LOGGER.info("Event published started successfully.")
    LOGGER.info("Publish TERCC event.")
    try:
        event = etos_library.events.send(tercc, links, data)
        await sync_to_async(etos_library.publisher.wait_for_unpublished_events)
    finally:
        await sync_to_async(etos_library.publisher.stop)
        await sync_to_async(etos_library.publisher.wait_close)
    LOGGER.info("Event published.")

    LOGGER.info("ETOS triggered successfully.")
    return {
        "tercc": event.meta.event_id,
        "artifact_id": artifact_id,
        "artifact_identity": identity,
        "event_repository": etos_library.debug.graphql_server,
    }
