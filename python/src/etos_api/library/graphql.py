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
"""Graphql query handler."""
import asyncio
from gql import gql, Client
from gql.transport.aiohttp import AIOHTTPTransport


class GraphqlQueryHandler:  # pylint:disable=too-few-public-methods
    """Handle Graphql queries."""

    def __init__(self, etos):
        """Initialize the async io transport.

        :param etos: ETOS Library instance.
        :type etos: :obj:`etos_lib.ETOS`
        """
        self.etos = etos
        self.transport = AIOHTTPTransport(
            url=self.etos.debug.graphql_server,
            timeout=self.etos.debug.default_http_timeout,
            client_session_args={"trust_env": True},
        )

    async def execute(self, query):
        """Execute a graphql query.

        :param query: Query to execute.
        :type query: str
        :return: Response from GraphQL.
        :rtype: dict
        """
        async with Client(
            transport=self.transport,
            fetch_schema_from_transport=True,
            execute_timeout=self.etos.debug.default_wait_timeout,
        ) as session:
            try:
                return await session.execute(gql(query))
            except asyncio.exceptions.TimeoutError:
                return None
