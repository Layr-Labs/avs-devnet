FROM golang:1.21 as build

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download && go mod tidy && go mod verify

COPY . .

WORKDIR /usr/src/app/operator/cmd
RUN go build -v -o /usr/local/bin/operator ./...

WORKDIR /usr/src/app

ENTRYPOINT [ "operator"]
CMD ["--config=/usr/src/app/config-files/operator.anvil.yaml"]