FROM golang:1.15.2-buster
LABEL maintainer="Rajdeep Bharati <rajdeep@buidllabs.io>"

RUN apt-get update
RUN apt-cache depends ffmpeg
RUN apt-get install -y ffmpeg
RUN ffmpeg -version
RUN apt-get install -y sqlite3

RUN mkdir /demuxapp
WORKDIR /demuxapp

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build cmd/main.go

RUN wget https://github.com/rajdeepbharati/go-livepeer/releases/download/v0.5.10-demux.1/livepeer -P ./livepeerPull/linux
RUN chmod +x ./livepeerPull/linux/livepeer

ENV IPFS_GATEWAY=${IPFS_GATEWAY}
ENV IPFS_REV_PROXY_ADDR=${IPFS_REV_PROXY_ADDR}
ENV LIVEPEER_COM_API_KEY=${LIVEPEER_COM_API_KEY}
ENV LIVEPEER_PRICING_TOOL=${LIVEPEER_PRICING_TOOL}
ENV PINATA_API_KEY=${PINATA_API_KEY}
ENV PINATA_SECRET_KEY=${PINATA_SECRET_KEY}
ENV POLL_INTERVAL=${POLL_INTERVAL}
ENV PORT=${PORT}
ENV POW_TOKEN=${POW_TOKEN}
ENV POWERGATE_ADDR=${POWERGATE_ADDR}
ENV TRUSTED_MINERS=${TRUSTED_MINERS}
ENV DEMUX_URL=${DEMUX_URL}
ENV MONGO_URI=${MONGO_URI}

EXPOSE ${PORT}

CMD [ "./main" ]
