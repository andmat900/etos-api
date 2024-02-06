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
"""Fake database library helpers."""
from queue import Queue
from threading import Event, RLock, Timer
from typing import Any, Iterator, Optional

from etcd3gw.lease import Lease

# pylint:disable=unused-argument


class FakeDatabase:
    """A fake database that follows the etcd client."""

    lock = RLock()

    def __init__(self):
        """Initialize fake reader and writer."""
        self.db_dict = {}
        self.expire = []
        self.leases = {}
        self.watchers = []

    def __call__(self):
        """Database instantiation faker."""
        return self

    def watch(self, path: str, range_end: Optional[str] = None) -> (Event, Iterator[dict]):
        """Watch for changes of a path."""
        canceled = Event()
        queue = Queue()

        def cancel():
            canceled.set()
            queue.put(None)

        def iterator():
            self.watchers.append(queue)
            try:
                while not canceled.is_set():
                    event = queue.get()
                    if event is None:
                        canceled.set()
                    if not canceled.is_set():
                        yield event
            finally:
                self.watchers.remove(queue)

        return iterator(), cancel

    def __event(self, event: dict) -> None:
        """Send an event to all watchers."""
        for watcher in self.watchers:
            watcher.put_nowait(event)

    def put(self, item: str, value: Any, lease: Optional[Lease] = None) -> None:
        """Put an item into database."""
        with self.lock:
            self.db_dict.setdefault(item, [])
            if lease is not None:
                self.leases[item] = lease
                timer = Timer(self.expire[lease.id], self.delete, args=(item,))
                timer.daemon = True
                timer.start()
            self.db_dict[item].append(str(value).encode())
        self.__event({"kv": {"key": item.encode(), "value": str(value).encode()}})

    def lease(self, ttl=30) -> Lease:
        """Create a lease."""
        with self.lock:
            self.expire.append(ttl)
        # ttl is unused since we do not actually make the post request that the regular
        # etcd client does. First argument to `Lease` is the ID that was returned by the
        # etcd server.
        return Lease(len(self.expire) - 1)

    def get(self, path: str) -> list[bytes]:
        """Get an item from database."""
        if isinstance(path, bytes):
            path = path.decode()
        with self.lock:
            return list(reversed(self.db_dict.get(path, [])))

    def get_prefix(self, prefix: str) -> list[tuple[bytes, dict]]:
        """Get items based on prefix."""
        if isinstance(prefix, bytes):
            prefix = prefix.decode()
        paths = []
        with self.lock:
            for key, value in self.db_dict.items():
                if key.startswith(prefix):
                    paths.append((value[-1], {"key": key.encode()}))
        return paths

    def delete(self, path: str) -> None:
        """Delete a single item."""
        with self.lock:
            del self.db_dict[path]
            if self.leases.get(path):
                self.expire.pop(self.leases.get(path).id)
                del self.leases[path]
        self.__event({"kv": {"key": path.encode()}, "type": "DELETE"})

    def delete_prefix(self, prefix: str) -> None:
        """Delete items based on prefix."""
        with self.lock:
            db_dict = self.db_dict.copy()
        for key in db_dict:
            if key.startswith(prefix):
                self.delete(key)
