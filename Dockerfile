FROM golang:1.15.2-buster
LABEL maintainer="Rajdeep Bharati <rajdeep@buidllabs.io>"

RUN apt-get update
RUN apt-cache depends ffmpeg
RUN apt-get install -y ffmpeg
RUN ffmpeg -version

RUN mkdir /demuxapp
WORKDIR /demuxapp

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build cmd/main.go

RUN curl https://build.livepeer.live/0.5.10-32544624/livepeer-linux-amd64.tar.gz | tar -xz
RUN cp livepeer-linux-amd64/livepeer livepeerPull/linux

ENV LIVEPEER_COM_API_KEY=${LIVEPEER_COM_API_KEY}
ENV LIVEPEER_PRICING_TOOL=${LIVEPEER_PRICING_TOOL}
ENV POWERGATE_ADDR=${POWERGATE_ADDR}
ENV IPFS_REV_PROXY_ADDR=${IPFS_REV_PROXY_ADDR}
ENV PINATA_API_KEY=${PINATA_API_KEY}
ENV PINATA_SECRET_KEY=${PINATA_SECRET_KEY}
ENV POLL_INTERVAL=${POLL_INTERVAL}
ENV PORT=${PORT}

EXPOSE ${PORT}

CMD [ "./main" ]
