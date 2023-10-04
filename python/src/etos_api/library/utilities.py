# Copyright 2020 Axis Communications AB.
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
"""ETOS API utilities."""
import asyncio
import sys
import functools
from inspect import iscoroutinefunction
from contextlib import asynccontextmanager


async def sync_to_async(function, *args, **kwargs):
    """Convert synchronous method to async, using threads.

    Credit for this function goes to https://github.com/django/asgiref
    and their SyncToAsync class.

    :param function: Function or method to call.
    :type function: function
    :param args: Positional arguments to function call.
    :type args: tuple
    :param kwargs: Keyword arguments to function call.
    :type kwargs: dict
    :return: Return value from function call.
    :rtype: Any
    """

    def thread_exception_handler(exception_info, function, *args, **kwargs):
        """Call function and handle its exceptions properly.

        :param exception_info: If there's an exception, run the function inside
                               the except block so that exc_info is populated correctly.
        :type exception_info: tuple
        :param function: Function or method to call.
        :type function: function
        :param args: Positional arguments to function call.
        :type args: tuple
        :param kwargs: Keyword arguments to function call.
        :type kwargs: dict
        :return: Return value from function call.
        :rtype: Any
        """
        if exception_info[1]:
            try:
                raise exception_info[1]
            except:  # pylint: disable=bare-except
                return function(*args, **kwargs)
        return function(*args, **kwargs)

    loop = asyncio.get_event_loop()
    future = loop.run_in_executor(
        None,  # Default executor.
        functools.partial(
            thread_exception_handler, sys.exc_info(), function, *args, **kwargs
        ),
    )
    return await asyncio.wait_for(future, timeout=None)


@asynccontextmanager
async def aclosing(thing):
    """Asynchronous closing context manager.

    :param thing: Thing to close.
    :type thing: Any
    """
    try:
        yield thing
    finally:
        if iscoroutinefunction(thing.close):
            await thing.close()
        else:
            await sync_to_async(thing.close)
