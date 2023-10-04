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
"""ETOS API selftest router."""
import logging
from starlette.responses import Response
from fastapi import APIRouter

ROUTER = APIRouter()
LOGGER = logging.getLogger(__name__)


@ROUTER.get("/selftest/ping", tags=["maintenance"], status_code=204)
async def ping():
    """Ping the ETOS service in order to check if it is up and running.

    :return: HTTP 204 response.
    :rtype: :obj:`starlette.responses.Response`
    """
    return Response(status_code=204)


@ROUTER.head("/selftest/ping", tags=["maintenance"], status_code=204)
async def head_ping():
    """Exists solely for backwards compatibility. DEPRECATED."""
    LOGGER.warning("DEPRECATED HEAD request to ping received!")
    return Response(status_code=204)
