FROM golang AS build

WORKDIR /src

COPY ./ /src/

RUN go build -o sai-interax-manager -buildvcs=false

FROM ubuntu

WORKDIR /srv
RUN apt-get update && apt-get -y install ca-certificates

COPY --from=build /src/sai-interax-manager /srv/

RUN chmod +x /srv/sai-interax-manager

CMD /srv/sai-interax-manager start > /srv/logs/app.log 2>&1
