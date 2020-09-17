# Demux

A gateway to facilitate a decentralised streaming ecosystem.

## Getting Started

- Clone the repo: `git clone https://github.com/buidl-labs/Demux`
- Download the latest build of livepeer pull mode:
  - linux: https://build.livepeer.live/0.5.10-32544624/livepeer-linux-amd64.tar.gz
  - mac: https://build.livepeer.live/0.5.10-32544624/livepeer-darwin-amd64.tar.gz
- Place the `livepeer` binary inside `Demux/livepeerPull/linux` or `Demux/livepeerPull/darwin` directory, depending on your OS.
- Create a file `.env` and set environment variables (sample present in `Demux/.env.sample`)
- Run your docker daemon.
- Build the docker image: `docker build --tag demux:latest .`
- Run Demux: `docker run -p 8000:8000 --env-file ./.env demux:latest`

## API Endpoints

- **`POST /assets`**

  This is used to upload a video for streaming.

  Sample request:

  ```bash
  $ curl http://localhost:8000/assets -F inputfile=@/Users/johndoe/hello.mp4
  ```

  Sample response:

  ```json
  {
    "AssetID": "fba8cda5-6c71-46d7-ac15-28424343c037"
  }
  ```

- **`GET /assets/<asset_id>`**

  This endpoint gives us the status of an asset (uploaded video) in the pipeline.

  Sample request:

  ```bash
  $ curl http://localhost:8000/assets/fba8cda5-6c71-46d7-ac15-28424343c037
  ```

  Sample response:

  ```json
  {
    "AssetID": "fba8cda5-6c71-46d7-ac15-28424343c037",
    "AssetStatus": 3,
    "CID": "bafybeiew6zbs4ljg37phxr3ejt5ydci2ir4nkcbuqdkxctvzyip6hb7one",
    "Expiry": 0,
    "Miner": "t01000",
    "Status": "Completed Filecoin storage deal",
    "StorageCost": 0.000001019146484375,
    "TranscodingCost": "2.838960491e+13"
  }
  ```

## Requirements

- go 1.14.4
- ffmpeg
