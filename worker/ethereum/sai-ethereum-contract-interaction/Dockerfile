FROM golang AS build

WORKDIR /src/

COPY ./ /src/

RUN go build -o sai-eth-interaction -buildvcs=false

FROM ubuntu

WORKDIR /srv

# Copy binary from build stage
COPY --from=build /src/sai-eth-interaction /srv/sai-eth-interaction

# Copy other files
#COPY ./config.yml /srv/config.yml

RUN chmod +x /srv/sai-eth-interaction

# Set command to run your binary
CMD /srv/sai-eth-interaction start
