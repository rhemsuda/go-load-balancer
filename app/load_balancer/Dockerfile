FROM golang:alpine

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

ARG PORT

WORKDIR /build

COPY go.mod .
RUN go mod download

COPY . .

RUN go build -o main .

WORKDIR /dist

RUN cp /build/main .

EXPOSE ${PORT}

CMD ["/dist/main"]
