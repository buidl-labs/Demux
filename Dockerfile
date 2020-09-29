FROM golang:1.15.2-buster
LABEL maintainer="Rajdeep Bharati <rajdeep@buidllabs.io>"

RUN apt-get update

RUN mkdir /demuxipfsrevproxy
WORKDIR /demuxipfsrevproxy

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build cmd/main.go

ENV PORT=${PORT}
ENV POW_IPFS=${POW_IPFS}

EXPOSE ${PORT}

CMD [ "./main" ]
