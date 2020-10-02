# Demux

[![Made by BUIDL Labs](https://img.shields.io/badge/made%20by-BUIDL%20Labs-informational.svg)](https://buidllabs.io)
[![Go Report Card](https://goreportcard.com/badge/github.com/buidl-labs/Demux)](https://goreportcard.com/report/github.com/buidl-labs/Demux)
[![GitHub action](https://github.com/buidl-labs/Demux/workflows/Tests/badge.svg)](https://github.com/buidl-labs/Demux/actions)

A gateway to facilitate a decentralised streaming ecosystem.

Workflow depicted below:
![image](https://user-images.githubusercontent.com/24296199/94940994-e923d080-04f1-11eb-8c3d-5aad1f31e91f.png)


Currently hosted at https://demux.onrender.com/. For authentication credentials, please reach out to us at [saumay@buidllabs.io](saumay@buidllabs.io).

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

  Required fields:

  - User authentication: `<TOKEN_ID>:<TOKEN_SECRET>` (colon-separated)
  - File `input_file`: local file path

  Response fields:

  ```json
   - "asset_id": type string
      - Used to identify an uploaded video.
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

  Response fields:

  ```json
   - "asset_error":                type boolean
      - Initially its value is false, it will become true in case there is an error.
   - "asset_id":                   type string
      - Used to identify an uploaded video.
   - "asset_ready":                type boolean
      - Initially its value is false, it will become true once the video is ready for streaming.
   - "asset_status":               type string
      - Can have five possible values, corresponding to `asset_status_code`:
          - 0: "video uploaded successfully"
          - 1: "processing in livepeer"
          - 2: "attempting to pin to ipfs"
          - 3: "pinned to ipfs, attempting to store in filecoin"
          - 4: "stored in filecoin"
   - "asset_status_code":          type int
      - Possible values: 0, 1, 2, 3, 4
   - "created_at":                 type int
      - Unix timestamp of asset creation.
   - "storage_cost":               type int
      - Actual storage cost in filecoin network (value in attoFIL).
   - "storage_cost_estimated":     type int
      - Estimated storage cost in filecoin network (value in attoFIL).
   - "stream_url":                 type string
      - URL to stream the video.
   - "transcoding_cost": type int
      - Actual transcoding cost in livepeer network (value in WEI).
   - "transcoding_cost_estimated": type int
      - Estimated transcoding cost in livepeer network (value in WEI).
  ```

- **`POST /pricing`**

  This is used to estimate the transcoding and storage cost for a given video.

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

  Required fields:

  ```json
   - "video_duration":   type int
      - Duration of the video in seconds. Its value must be greater than `0`.
   - "video_file_size":  type int
      - Size of the video in MiB (1 MiB = 2^20 B).
   - "storage_duration": type int
      - Duration in seconds for which you want to store the video stream in filecoin. Its value must be between `2628003` and `315360000`.
  ```

  Response fields:

  ```json
   - "storage_cost_estimated":     type int
      - Estimated storage cost in filecoin network (value in attoFIL).
   - "transcoding_cost_estimated": type int
      - Estimated transcoding cost in livepeer network (value in WEI).
  ```

## Requirements

- go 1.15.2
- ffmpeg
