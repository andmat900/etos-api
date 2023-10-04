# Copyright 2021 Axis Communications AB.
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
"""ETOS API context based logging."""
import logging
from contextvars import ContextVar
from etos_lib.logging.logger import FORMAT_CONFIG


class ContextLogging(logging.Logger):
    """A specialized context based logging class.

    The ETOS API is based on asyncio and does not have
    threaded contexts when running requests in parallel
    and as such it cannot use the threading.local that is
    used within FORMAT_CONFIG in the ETOS library.

    This context based logging module replaces the FORMAT_CONFIG
    with a ContextVar instead, which works with asyncio, and calls
    get for each logging method called.
    """

    identifier = ContextVar("identifier")

    def critical(self, msg, *args, **kwargs):
        """Add identifier to critical calls.

        For documentation read :obj:`logging.Logger.critical`
        """
        FORMAT_CONFIG.identifier = self.identifier.get("Main")  # Default=Main
        return super().critical(msg, *args, **kwargs)

    def error(self, msg, *args, **kwargs):
        """Add identifier to error calls.

        For documentation read :obj:`logging.Logger.error`
        """
        FORMAT_CONFIG.identifier = self.identifier.get("Main")  # Default=Main
        return super().error(msg, *args, **kwargs)

    def warning(self, msg, *args, **kwargs):
        """Add identifier to warning calls.

        For documentation read :obj:`logging.Logger.warning`
        """
        FORMAT_CONFIG.identifier = self.identifier.get("Main")  # Default=Main
        return super().warning(msg, *args, **kwargs)

    def info(self, msg, *args, **kwargs):
        """Add identifier to info calls.

        For documentation read :obj:`logging.Logger.info`
        """
        FORMAT_CONFIG.identifier = self.identifier.get("Main")  # Default=Main
        return super().info(msg, *args, **kwargs)

    def debug(self, msg, *args, **kwargs):
        """Add identifier to debug calls.

        For documentation read :obj:`logging.Logger.debug`
        """
        FORMAT_CONFIG.identifier = self.identifier.get("Main")  # Default=Main
        return super().debug(msg, *args, **kwargs)


logging.setLoggerClass(ContextLogging)
