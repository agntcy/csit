FROM ghcr.io/spiffe/spire-server:1.14.5 AS spire-server
FROM alpine:3.20

COPY --from=spire-server /opt/spire/bin/spire-server /opt/spire/bin/spire-server

COPY spire-server-token-generate.sh /spire-server-token-generate.sh
COPY spire-server-entry-create.sh /spire-server-entry-create.sh

RUN chmod +x /spire-server-token-generate.sh
RUN chmod +x /spire-server-entry-create.sh

ENTRYPOINT ["/spire-server-token-generate.sh"]
