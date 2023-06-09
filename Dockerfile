FROM python:3.9.0-buster AS build

COPY . /src
WORKDIR /src
RUN python3 setup.py bdist_wheel

FROM python:3.9.0-slim-buster
ARG TZ
ENV TZ=$TZ

COPY --from=build /src/dist/*.whl /tmp

# hadolint ignore=DL3013
# hadolint ignore=DL3008
RUN apt-get update && \
    apt-get install -y gcc libc-dev tzdata --no-install-recommends && \
    pip install --no-cache-dir /tmp/*.whl && \
    apt-get purge -y --auto-remove gcc libc-dev && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -r etos && useradd -r -m -s /bin/false -g etos etos
USER etos
EXPOSE 8080

LABEL org.opencontainers.image.source=https://github.com/eiffel-community/etos-api
LABEL org.opencontainers.image.authors=etos-maintainers@googlegroups.com
LABEL org.opencontainers.image.licenses=Apache-2.0

ENTRYPOINT ["uvicorn", "etos_api.main:APP", "--host=0.0.0.0", "--port=8080"]
