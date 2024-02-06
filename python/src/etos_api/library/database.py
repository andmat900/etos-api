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
"""ETCD helpers."""
import os
from threading import Event
from typing import Any, Iterator, Optional, Union

from etcd3gw import client
from etos_lib.lib.config import Config as ETOSConfig


class ETCDPath:
    """An ETCD path is like a filesystem path, but it works with keys in ETCD."""

    def __init__(self, path: Union[str, bytes] = "/") -> None:
        """Initialize."""
        if ETOSConfig().get("database") is None:
            ETOSConfig().set(
                "database",
                client(
                    host=os.getenv("ETOS_ETCD_HOST", "etcd-client"),
                    port=int(os.getenv("ETOS_ETCD_PORT", "2379")),
                ),
            )
        self.database: client = ETOSConfig().get("database")
        if isinstance(path, bytes):
            path = path.decode()
        self.path = path

    def join(self, new: str) -> "ETCDPath":
        """Join this path with another path.

        :param new: New child path 'below' current.
        """
        if new.startswith("/"):
            new = new[1:]
        return ETCDPath("/".join((self.path, new)))

    def write(self, value: Any, expire: Optional[int] = None) -> None:
        """Write a value to an ETCD path.

        :param value: Value to write to database.
        :param expire: Optional expiration time in seconds.
        """
        lease = None
        if expire is not None:
            lease = self.database.lease(expire)
        self.database.put(self.path, value, lease)

    def read(self) -> Optional[bytes]:
        """Read the values from an ETCD path."""
        try:
            return self.database.get(self.path)[0]
        except IndexError:
            return None

    def read_all(self) -> list[tuple[bytes, dict]]:
        """Read values of all keys "below" a path."""
        return self.database.get_prefix(self.path)

    def watch(self) -> tuple[Event, Iterator[dict]]:
        """Watch an ETCD path for any changes."""
        return self.database.watch(self.path)

    def watch_all(self) -> tuple[Event, Iterator[dict]]:
        """Watch an ETCD path for any changes to itself or its children."""
        return self.database.watch(self.path, range_end="\0")

    def delete(self) -> None:
        """Delete the ETCD path."""
        self.database.delete(self.path)

    def delete_all(self) -> None:
        """Delete the ETCD path and paths "below"."""
        self.database.delete_prefix(self.path)

    def __str__(self) -> str:
        """Represent the ETCD path as a string."""
        return self.path

    def __repr__(self) -> str:
        """Represent the ETCD path as a string."""
        return self.path
