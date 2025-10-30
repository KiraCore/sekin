# Build
FROM golang AS build

WORKDIR /src

COPY ./ /src/

RUN go build -o sai-cosmos-indexer -buildvcs=false

FROM ubuntu

WORKDIR /srv
RUN apt-get update && apt-get -y install ca-certificates

# Copy binary from build stage
COPY --from=build /src/sai-cosmos-indexer /srv/

RUN chmod +x /srv/sai-cosmos-indexer

# Set command to run your binary
CMD /srv/sai-cosmos-indexer start
