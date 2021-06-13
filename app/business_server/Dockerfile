FROM golang:alpine

# ARG PORT

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o main .

WORKDIR /dist

RUN cp /build/main .

EXPOSE ${PORT}

# RUN echo "#!/bin/sh\n exec /dist/main \"$PORT\"" > ./entrypoint.sh
# RUN chmod +x ./entrypoint.sh

# ENTRYPOINT ["./entrypoint.sh"]

CMD [ "/dist/main" ]

# CMD ["/bin/sh", "-c", "exec /dist/main", "echo ${PORT}"]

