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
"""ETOS API."""
import logging

from fastapi import FastAPI

# from opentelemetry.sdk.trace import TracerProvider
from starlette.responses import RedirectResponse

from etos_api import routers

APP = FastAPI()
LOGGER = logging.getLogger(__name__)


@APP.post("/")
async def redirect_post_to_root():
    """Redirect post requests to root to the start ETOS endpoint.

    :return: Redirect to etos.
    :rtype: :obj:`starlette.responses.RedirectResponse`
    """
    LOGGER.debug("Redirecting post requests to the root endpoint to '/etos'")
    # DEPRECATED. Exists only for backwards compatibility.
    return RedirectResponse(url="/etos", status_code=308)  # 308 = Permanent Redirect


@APP.head("/")
async def redirect_head_to_root():
    """Redirect head requests to root to the selftest/ping endpoint.

    :return: Redirect to selftest/ping.
    :rtype: :obj:`starlette.responses.RedirectResponse`
    """
    LOGGER.debug("Redirecting head requests to the root endpoint to '/sefltest/ping'")
    # DEPRECATED. Exists only for backwards compatibility.
    return RedirectResponse(url="/selftest/ping", status_code=308)  # 308 = Permanent Redirect


APP.include_router(routers.etos.ROUTER)
APP.include_router(routers.selftest.ROUTER)
APP.include_router(routers.logs.ROUTER)
