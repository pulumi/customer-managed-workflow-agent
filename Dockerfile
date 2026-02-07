FROM alpine:3.20

ARG DOCKER_VERSION
ENV PATH="${PATH}:/usr/local/bin"
ENV DOCKER_VERSION="${DOCKER_VERSION}"

# Goreleaser will copy the release binaries to the root directory build context during a release.
COPY customer-managed-workflow-agent /usr/local/bin/customer-managed-workflow-agent
COPY workflow-runner /usr/local/bin/workflow-runner

CMD [ "customer-managed-workflow-agent", "run"]
