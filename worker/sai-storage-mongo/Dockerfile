# Build stage
FROM golang AS build

WORKDIR /src/

COPY ./ /src/

RUN go build -o sai-storage-bin -buildvcs=false

FROM ubuntu

WORKDIR /srv

# Copy binary from build stage
COPY --from=build /src/sai-storage-bin /srv/sai-storage-bin

# Copy other files
#COPY ./config.yml /srv/config.yml

RUN chmod +x /srv/sai-storage-bin

# Set command to run your binary
CMD /srv/sai-storage-bin start

EXPOSE 8880
