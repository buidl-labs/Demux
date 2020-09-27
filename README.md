# Demux

[![Made by BUIDL Labs](https://img.shields.io/badge/made%20by-BUIDL%20Labs-informational.svg)](https://buidllabs.io)
[![Go Report Card](https://goreportcard.com/badge/github.com/buidl-labs/Demux)](https://goreportcard.com/report/github.com/buidl-labs/Demux)
[![GitHub action](https://github.com/buidl-labs/Demux/workflows/Tests/badge.svg)](https://github.com/buidl-labs/Demux/actions)

A gateway to facilitate a decentralised streaming ecosystem.

## Getting Started

- Clone the repo: `git clone https://github.com/buidl-labs/Demux`

### Using Docker

- Create a file `.env` and set environment variables (sample present in `Demux/.env.sample`)
- Run your docker daemon.
- Build the docker image: `docker build --tag demux:latest .`
- Run Demux: `docker run -p 8000:8000 --env-file ./.env demux:latest`

### Without Docker

- Download the latest build of livepeer pull mode:
  - linux: https://build.livepeer.live/0.5.10-32544624/livepeer-linux-amd64.tar.gz
  - mac: https://build.livepeer.live/0.5.10-32544624/livepeer-darwin-amd64.tar.gz
- Place the `livepeer` binary inside `Demux/livepeerPull/linux` or `Demux/livepeerPull/darwin` directory, depending on your OS.
- Make sure you have golang and ffmpeg installed.
- Set environment variables (sample present in `Demux/.env.sample`)
- Run Demux: `make run`

## API Endpoints

- **`POST /asset`**

  This is used to upload a video for streaming.

  Sample request:

  ```bash
  $ curl http://localhost:8000/asset -u <TOKEN_ID>:<TOKEN_SECRET> -F input_file=@/Users/johndoe/hello.mp4
  ```

  Sample response:

  ```json
  {
    "asset_id": "fba8cda5-6c71-46d7-ac15-28424343c037"
  }
  ```

- **`GET /asset/<asset_id>`**

  This endpoint gives us the status of an asset (uploaded video) in the pipeline.

  Sample request:

  ```bash
  $ curl http://localhost:8000/asset/fba8cda5-6c71-46d7-ac15-28424343c037
  ```

  Sample response:

  ```json
  {
    "asset_error": false,
    "asset_id": "595a040d-eb9c-4e71-97bc-ae2eba60fe61",
    "asset_ready": true,
    "asset_status": "pinned to ipfs, attempting to store in filecoin",
    "asset_status_code": 3,
    "created_at": 1601220874,
    "storage_cost": 0,
    "storage_cost_estimated": 6158801662122543,
    "stream_url": "https://ipfs.io/ipfs/bafybeied3ahoc2wm752myjiamod6tzvyujvxtikpl2pcy2it4loyiu63ni/root.m3u8",
    "transcoding_cost": 0,
    "transcoding_cost_estimated": 28389604913585
  }
  ```

- **`POST /pricing`**

  This is used to calculate the price of transcoding and storage for a given video.
  `storage_duration`: Duration in seconds for which you want to store the video stream in filecoin. Its value must be between `2628003` and `315360000`.
  `storage_cost_estimated` is in attoFIL and `transcoding_cost_estimated` is in WEI.

  Sample request:

  ```bash
  $ curl http://localhost:8000/pricing -F input_file=@/Users/johndoe/hello.mp4 -F storage_duration=3000000
  ```

  Sample response:

  ```json
  {
    "storage_cost_estimated": 5560825118337973,
    "transcoding_cost_estimated": 107880498671623
  }
  ```

## Requirements

- go 1.15.2
- ffmpeg
