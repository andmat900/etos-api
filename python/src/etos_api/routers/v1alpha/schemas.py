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
"""Schemas for the testrun endpoint."""

import os
from typing import Optional, Union
from uuid import UUID

# Pylint refrains from linting C extensions due to arbitrary code execution.
from pydantic import (
    BaseModel,
    Field,
    field_validator,
    model_validator,
)  # pylint:disable=no-name-in-module

# pylint: disable=too-few-public-methods
# pylint: disable=no-self-argument


class TestrunRequest(BaseModel):
    """Base class for testrun request models."""


class TestrunResponse(BaseModel):
    """Base class for testrun response models."""


class StartTestrunRequest(TestrunRequest):
    """Request model for the start endpoint of the ETOS testrun API."""

    artifact_identity: Optional[str]
    artifact_id: Optional[UUID] = Field(default=None, validate_default=True)
    test_suite_url: str
    dataset: Union[dict, list[dict]] = {}
    execution_space_provider: Optional[str] = os.getenv("DEFAULT_EXECUTION_SPACE_PROVIDER", "")
    iut_provider: Optional[str] = os.getenv("DEFAULT_IUT_PROVIDER", "")
    log_area_provider: Optional[str] = os.getenv("DEFAULT_LOG_AREA_PROVIDER", "")
    timeout: Optional[int] = None
    deadline: Optional[int] = None

    @model_validator(mode="after")
    def validate_timeout_or_deadline(self):
        """Validate that only one of timeout or deadline is set.

        The controller will ignore timeout if deadline is set, so we should
        prevent users from setting both to avoid confusion.

        :return: The validated model.
        :rtype: StartTestrunRequest
        """
        if self.timeout is not None and self.deadline is not None:
            raise ValueError("Only one of 'timeout' or 'deadline' can be set, not both.")
        return self

    @field_validator("artifact_id")
    def validate_id_or_identity(cls, artifact_id, info):
        """Validate that id/identity is set correctly.

        :param artifact_id: The value of 'artifact_id' to validate.
        :value artifact_id: str or None
        :param info: The information about the model.
        :type info: FieldValidationInfo
        :return: The value of artifact_id.
        :rtype: str or None
        """
        values = info.data
        artifact_identity = values.get("artifact_identity")

        # Treat empty/whitespace-only identity as not provided
        if isinstance(artifact_identity, str) and not artifact_identity.strip():
            artifact_identity = None

        # Check that at least one is provided
        if artifact_identity is None and not artifact_id:
            raise ValueError("Missing or invalid identity: provide a valid UUID or PackageURL.")

        # Check that only one is provided
        if artifact_identity is not None and artifact_id:
            raise ValueError("Only one of 'artifact_identity' or 'artifact_id' is required.")

        # Validate artifact_identity format if provided
        if artifact_identity is not None:
            if not isinstance(artifact_identity, str) or not artifact_identity.startswith("pkg:"):
                raise ValueError("Invalid artifact_identity: must be a valid PackageURL.")

        # Note: artifact_id UUID validation is handled by Pydantic's built-in UUID type validation

        return artifact_id


class StartTestrunResponse(TestrunResponse):
    """Response model for the start endpoint of the ETOS testrun API."""

    event_repository: str
    tercc: UUID
    artifact_id: UUID
    artifact_identity: str


class AbortTestrunResponse(TestrunResponse):
    """Response model for the abort endpoint of the ETOS testrun API."""

    message: str
