FROM golang:1.22.3 AS shidai-builder

WORKDIR /app

ARG VERSION

ENV CGO_ENABLED=0 \
		GOOS=linux \
		GOARCH=amd64 \


COPY ./src/shidai/go.* /app

RUN go mod download

COPY /src/shidai /app

RUN go build -a -tags netgo -installsuffix cgo ldflags="-X main.Version=${VERSION}"-o /shidai /app/cmd/main.go

FROM scratch

COPY --from=shidai-builder /shidai /shidai

CMD ["/shidai", "start"]

EXPOSE 8282
