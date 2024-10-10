FROM golang:1.23

RUN go install github.com/Layr-Labs/eigensdk-go/cmd/egnkey@latest

ENTRYPOINT "egnkey"
