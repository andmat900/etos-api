# Copyright 2020-2022 Axis Communications AB.
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
"""Schemas for the environment provider endpoint."""
from typing import Union
from pydantic import BaseModel


class ConfigureEnvironmentProviderRequest(BaseModel):
    """Model for the configure environment provider API."""

    suite_id: str
    dataset: Union[dict, list]
    execution_space_provider: str
    iut_provider: str
    log_area_provider: str
