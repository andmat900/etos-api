# Copyright Axis Communications AB.
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
"""ETOS API log handler."""
import os
import asyncio
import logging
from uuid import UUID
from kubernetes import client, config
from fastapi import APIRouter, HTTPException

from sse_starlette.sse import EventSourceResponse
from starlette.requests import Request
import httpx

NAMESPACE_FILE = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
LOGGER = logging.getLogger(__name__)
ROUTER = APIRouter()

try:
    config.load_incluster_config()
except config.ConfigException:
    try:
        config.load_config()
    except config.ConfigException:
        LOGGER.warning("Could not load a Kubernetes config")


def namespace() -> str:
    """Get current namespace if available."""
    if not os.path.isfile(NAMESPACE_FILE):
        LOGGER.warning(
            "Not running in Kubernetes. Cannot figure out namespace. "
            "Defaulting to environment variable 'ETOS_NAMESPACE'."
        )
        return os.getenv("ETOS_NAMESPACE")
    with open(NAMESPACE_FILE, encoding="utf-8") as namespace_file:
        return namespace_file.read()


@ROUTER.get("/logs/{uuid}", tags=["logs"])
async def get_logs(uuid: UUID, request: Request):
    """Get logs from an ETOS pod and stream them back as server sent events."""
    LOGGER.identifier.set(str(uuid))
    corev1 = client.CoreV1Api()
    thread = corev1.list_namespaced_pod(namespace(), async_req=True)
    pod_list = thread.get()

    ip_addr = None
    for pod in pod_list.items:
        if pod.status.phase == "Running" and pod.metadata.name.startswith(
            f"suite-runner-{str(uuid)}"
        ):
            ip_addr = pod.status.pod_ip
    if ip_addr is None:
        raise HTTPException(
            status_code=404, detail=f"Suite runner with UUID={uuid} not found"
        )

    async def sse(url):
        index = 0
        while True:
            if await request.is_disconnected():
                break
            try:
                response = httpx.get(url)
                lines = response.text.splitlines()
                for message in lines[index:]:
                    LOGGER.debug(message)
                    yield {"id": index + 1, "event": "message", "data": message}
                    index += 1
            except httpx.RemoteProtocolError:
                LOGGER.exception("Failed to connect to pod %r", url)
            except IndexError:
                pass
            await asyncio.sleep(1)

    return EventSourceResponse(
        sse(f"http://{ip_addr}:8000/log"),
    )
