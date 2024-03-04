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
"""Register providers."""
import json
import logging
import os
from pathlib import Path
from typing import Iterator

import jsonschema

from ..database import ETCDPath
from .execution_space_provider import execution_space_provider_schema
from .iut_provider import iut_provider_schema
from .log_area_provider import log_area_provider_schema


class RegisterProviders:  # pylint:disable=too-few-public-methods
    """Register one or several new providers to the environment provider."""

    logger = logging.getLogger(__name__)

    def __init__(self) -> None:
        """Load providers."""
        self.logger.info("Registering environment providers")
        self.root = ETCDPath("/environment/provider")
        self.load_providers_from_disk()

    def load_providers_from_disk(self) -> None:
        """Register provider files from file system, should environment variables be set."""

        if os.getenv("EXECUTION_SPACE_PROVIDERS"):
            for provider in self.providers(Path(os.getenv("EXECUTION_SPACE_PROVIDERS"))):
                self.logger.info("Registering execution space provider: %s", provider)
                self.validate(provider, execution_space_provider_schema(provider))
                self.root.join(f"execution-space/{provider['execution_space']['id']}").write(
                    provider
                )

        if os.getenv("LOG_AREA_PROVIDERS"):
            for provider in self.providers(Path(os.getenv("LOG_AREA_PROVIDERS"))):
                self.logger.info("Registering log area provider: %s", provider)
                self.validate(provider, log_area_provider_schema(provider))
                self.root.join(f"log-area/{provider['log']['id']}").write(
                    provider
                )

        if os.getenv("IUT_PROVIDERS"):
            for provider in self.providers(Path(os.getenv("IUT_PROVIDERS"))):
                self.logger.info("Registering IUT provider: %s", provider)
                self.validate(provider, iut_provider_schema(provider))
                self.root.join(f"iut/{provider['iut']['id']}").write(
                    provider
                )

    def providers(self, directory: Path) -> Iterator[dict]:
        """Read provider json files from a directory.

        :param directory: Directory to read provider json files from.
        :return: An iterator of the json files.
        """
        try:
            filenames = os.listdir(directory)
        except FileNotFoundError:
            return
        for provider_filename in filenames:
            if not directory.joinpath(provider_filename).is_file():
                self.logger.warning("Not a file: %r", provider_filename)
                continue
            with directory.joinpath(provider_filename).open() as provider_file:
                self.logger.info("Reading provider file: %s", provider_filename)
                yield json.load(provider_file)

    def validate(self, provider: dict, schema: str) -> dict:
        """Validate a provider JSON against schema.

        :param provider: Provider JSON to validate.
        :param schema: JSON schema to validate against.
        :return: Provider JSON that was validated.
        """
        self.logger.debug("Validating provider %r against %r", provider, schema)
        with open(schema, encoding="UTF-8") as schema_file:
            schema = json.load(schema_file)
        jsonschema.validate(instance=provider, schema=schema)
        return provider
