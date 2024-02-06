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
"""Environment for ETOS testruns."""
import json
from collections import OrderedDict
from typing import Optional, Union

from pydantic import BaseModel  # pylint:disable=no-name-in-module

from .database import ETCDPath


class Configuration(BaseModel):
    """Model for the ETOS testrun configuration."""

    suite_id: str
    dataset: Union[dict, list]
    execution_space_provider: str
    iut_provider: str
    log_area_provider: str


async def configure_testrun(configuration: Configuration) -> None:
    """Configure an ETOS testrun with the configuration passed by user.

    :param configuration: The configuration to save.
    """
    testrun = ETCDPath(f"/testrun/{configuration.suite_id}")
    providers = ETCDPath("/environment/provider")

    await do_configure(
        providers.join(f"log-area/{configuration.log_area_provider}"),
        configuration.log_area_provider,
        testrun.join("provider/log-area"),
    )
    await do_configure(
        providers.join(f"execution-space/{configuration.execution_space_provider}"),
        configuration.execution_space_provider,
        testrun.join("provider/execution-space"),
    )
    await do_configure(
        providers.join(f"iut/{configuration.iut_provider}"),
        configuration.iut_provider,
        testrun.join("provider/iut"),
    )
    await save_json(testrun.join("provider/dataset"), json.dumps(configuration.dataset))


async def do_configure(path: ETCDPath, provider_id: str, testrun: ETCDPath) -> None:
    """Configure a provider based on provider ID and save it to a testrun.

    :param path: Path to load provider from.
    :param provider_id: The ID of the provider to load.
    :param testrun: Where to store the loaded provider.
    """
    if (provider := await load(path)) is None:
        raise AssertionError(f"{provider_id} does not exist")
    await save_json(testrun, provider)


async def load(path: ETCDPath) -> Optional[dict]:
    """Load a provider from an ETCD path.

    :param path: Path to load data from. Will assume it's JSON and load is as such.
    """
    provider = path.read()
    if provider:
        return json.loads(provider, object_pairs_hook=OrderedDict)
    return None


async def save_json(path: ETCDPath, data: dict, expire=3600) -> None:
    """Save data as json to an ETCD path.

    :param path: The path to store data on.
    :param data: The data to save. Will be dumped to JSON before saving.
    :param expire: How long, in seconds, to set the expiration to.
    """
    path.write(json.dumps(data), expire=expire)
