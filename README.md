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

## API Endpoints

- **`POST /upload`**

  This is used to upload a video for streaming.

  Sample request:

  ```bash
  $ curl http://localhost:8000/upload -u <TOKEN_ID>:<TOKEN_SECRET> -F input_file=@/Users/johndoe/hello.mp4
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

  This is used to estimate the transcoding and storage cost for a given video.

  Input data params:

  - `video_duration`: Duration of the video in seconds. Its value must be greater than `0`.
  - `video_file_size`: Size of the video in MiB (`1 MiB = 2^20 B`).
  - `storage_duration`: Duration in seconds for which you want to store the video stream in filecoin. Its value must be between `2628003` and `315360000`.

  Output:
  `storage_cost_estimated` is in attoFIL and `transcoding_cost_estimated` is in WEI.

  Sample request:

  ```bash
  $ curl http://localhost:8000/pricing -d "{\"video_duration\":60, \"video_file_size\":28, \"storage_duration\":2628005}"
  ```

  Sample response:

  ```json
  {
    "storage_cost_estimated": 1562410068450,
    "transcoding_cost_estimated": 170337629481511
  }
  ```

## Requirements

- go 1.15.2
- ffmpeg
