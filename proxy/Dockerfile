FROM golang AS build

WORKDIR /src

COPY ./ /src/

RUN go build -o sai-interax-proxy -buildvcs=false

FROM ubuntu

WORKDIR /srv
RUN apt-get update && apt-get -y install ca-certificates

COPY --from=build /src/sai-interax-proxy /srv/

RUN chmod +x /srv/sai-interax-proxy

CMD /srv/sai-interax-proxy start > /srv/logs/app.log 2>&1
