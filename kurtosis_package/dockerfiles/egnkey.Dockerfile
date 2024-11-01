FROM golang:1.23

RUN go install github.com/Layr-Labs/eigensdk-go/cmd/egnkey@565bb44

ENTRYPOINT "egnkey"
