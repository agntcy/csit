# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

FROM alpine/helm:3

RUN apk add curl kubectl
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/bin

COPY Taskfile.yml /apps/Taskfile.yml
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
