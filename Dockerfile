# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
ARG ENABLE_PROXY=false
ARG BASE_IMAGE_TAG=nonroot
ARG GO_BUILD_IMAGE=golang:1.20

FROM --platform=$BUILDPLATFORM ${GO_BUILD_IMAGE} AS build-env
LABEL maintainer="gxthrj@163.com"

ARG ENABLE_PROXY
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG GIT_SHA=unknown

WORKDIR /build
COPY go.* ./

RUN if [ "$ENABLE_PROXY" = "true" ] ; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS="$TARGETOS" GOARCH="$TARGETARCH" make build \
    VERSION="$VERSION" GITSHA="$GIT_SHA" OSNAME="$TARGETOS" OSARCH="$TARGETARCH"

FROM gcr.io/distroless/static-debian12:${BASE_IMAGE_TAG}
LABEL maintainer="gxthrj@163.com"
ARG VERSION=dev
ARG GIT_SHA=unknown
LABEL org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$GIT_SHA" \
      org.opencontainers.image.source="https://github.com/goodrain/apisix-ingress-controller"
ENV TZ=Hongkong

WORKDIR /ingress-apisix
COPY --from=build-env /build/apisix-ingress-controller .
COPY ./conf/apisix-schema.json ./conf/apisix-schema.json

ENTRYPOINT ["/ingress-apisix/apisix-ingress-controller", "ingress", "--config-path", "/ingress-apisix/conf/config.yaml"]
