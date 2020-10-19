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

<br>

## API Endpoints

### At a Glance

- [`POST /asset`](#post-asset)
- [`POST /fileupload/<asset_id>`](#post-fileuploadasset_id)
- [`GET /upload/<asset_id>`](#get-uploadasset_id)
- [`GET /asset/<asset_id>`](#get-assetasset_id)
- [`POST /pricing`](#post-pricing)

## <br>

### `POST /asset`

Used to create a new _asset_. It returns an `asset_id` and a `url`, where the client can upload a file.

**Examples**

- Request:

  ```bash
  $ curl http://localhost:8000/asset -u <TOKEN_ID>:<TOKEN_SECRET> -d {}
  ```

  Basic authentication must be done by passing the user flag as shown above (colon-separated), or by passing the `Authorization` header.

  For example, in JavaScript:

  ```js
  headers: {
    'Authorization' : 'Basic ' + Buffer.from('<TOKEN_ID>:<TOKEN_SECRET>', 'utf-8').toString('base64')
  }
  ```

- Response:

  ```json
  {
    "asset_id": "34d048bb-1076-4937-8b91-6bcda7a6187c",
    "url": "http://localhost:8000/fileupload/34d048bb-1076-4937-8b91-6bcda7a6187c"
  }
  ```

<br>

### `POST /fileupload/<asset_id>`

This endpoint is used to upload a video in chunks using [resumable.js](https://github.com/23/resumable.js).

A frontend client can send the request in the following manner:

```js
targetURL = `http://localhost:8000/fileupload/34d048bb-1076-4937-8b91-6bcda7a6187c`
const r = new Resumable({
  target: targetURL
})
r.addFile(myfile) // myfile is the file (mp4 video) that is being uploaded.
```

Here, the `targetUrl` is the url received after creating an asset using `POST /asset`.
The status of the upload can be tracked using event listeners:

```js
r.on('fileAdded', function (file) {
  r.upload()
})

r.on('progress', function () {
  console.log(Math.floor(r.progress() * 100))
})

r.on('fileSuccess', function (file, message) {
  console.log('Successfully uploaded', file, 'message:', message)
})

r.on('fileError', function (file, message) {
  console.log('Error uploading the file:', message)
})
```

For more details, refer to the [resumable.js docs](https://github.com/23/resumable.js).

<br>

### `GET /upload/<asset_id>`

This endpoint lets us see the status of an upload.

**Examples**

- Request:

  ```bash
  $ curl http://localhost:8000/upload/34d048bb-1076-4937-8b91-6bcda7a6187c
  ```

- Response:

  ```json
  {
    "asset_id": "34d048bb-1076-4937-8b91-6bcda7a6187c",
    "error": false,
    "status": true,
    "url": "http://localhost:8000/fileupload/34d048bb-1076-4937-8b91-6bcda7a6187c"
  }
  ```

  In the response, `status` is initially `false`, and it becomes `true` when the file has been uploaded successfully to Demux (using `/POST /fileupload/<asset_id>`). `error` is `false` by default, and becomes `true` if there is some problem in uploading the file. A frontend client may poll for the upload status to change to `true` before proceeding to the next step in the workflow.

<br>

### `GET /asset/<asset_id>`

This endpoint gives us the status of an asset (uploaded video) in the pipeline.

**Examples**

- Request:

  ```bash
  $ curl http://localhost:8000/asset/34d048bb-1076-4937-8b91-6bcda7a6187c
  ```

- Response:

  ```json
  {
    "asset_error": false,
    "asset_id": "34d048bb-1076-4937-8b91-6bcda7a6187c",
    "asset_ready": true,
    "asset_status": "pinned to ipfs, attempting to store in filecoin",
    "asset_status_code": 3,
    "created_at": 1602077805,
    "storage_cost": 0,
    "storage_cost_estimated": 854019799804687,
    "stream_url": "https://demuxipfsrevproxy.onrender.com/ipfs/bafybeiddn2lzoybioi6xh76j7aa67jgxgyyxa42nirfxpo5q477432jzz4/root.m3u8",
    "thumbnail": "https://demuxipfsrevproxy.onrender.com/ipfs/bafybeiddn2lzoybioi6xh76j7aa67jgxgyyxa42nirfxpo5q477432jzz4/thumbnail.png",
    "transcoding_cost": 0,
    "transcoding_cost_estimated": 107880498671623
  }
  ```

**Fields**

- Response:

  | Field Name                 | Type      | Description                                                                                                                                                                                                                                                                                           |
  | -------------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
  | asset_error                | `boolean` | Initially its value is false, it will become true in case there is an error.                                                                                                                                                                                                                          |
  | asset_id                   | `string`  | Used to identify an uploaded video.                                                                                                                                                                                                                                                                   |
  | asset_ready                | `boolean` | Initially its value is false, it will become true once the video is ready for streaming.                                                                                                                                                                                                              |
  | asset_status               | `string`  | Can have five possible values, corresponding to `asset_status_code` <br> • -1: "asset created"<br> • 0: "video uploaded successfully"<br> • 1: "processing in livepeer"<br> • 2: "attempting to pin to ipfs"<br> • 3: "pinned to ipfs, attempting to store in filecoin"<br> • 4: "stored in filecoin" |
  | asset_status_code          | `int`     | Possible values: 0, 1, 2, 3, 4                                                                                                                                                                                                                                                                        |
  | created_at                 | `int`     | Unix timestamp of asset creation.                                                                                                                                                                                                                                                                     |
  | storage_cost               | `int`     | Actual storage cost in filecoin network (value in attoFIL).                                                                                                                                                                                                                                           |
  | storage_cost_estimated     | `int`     | Estimated storage cost in filecoin network (value in attoFIL).                                                                                                                                                                                                                                        |
  | stream_url                 | `string`  | URL to stream the video.                                                                                                                                                                                                                                                                              |
  | thumbnail                  | `string`  | Thumbnail for the video.                                                                                                                                                                                                                                                                              |
  | transcoding_cost           | `int`     | Actual transcoding cost in livepeer network (value in WEI).                                                                                                                                                                                                                                           |
  | transcoding_cost_estimated | `int`     | Estimated transcoding cost in livepeer network (value in WEI).                                                                                                                                                                                                                                        |

<br>

### `POST /pricing`

This is used to estimate the transcoding and storage cost for a given video.

**Examples**

- Request:

  ```bash
  $ curl http://localhost:8000/pricing -d "{\"video_duration\":60, \"video_file_size\":28, \"storage_duration\":2628005}"
  ```

- Response:

  ```json
  {
    "storage_cost_estimated": 1562410068450,
    "transcoding_cost_estimated": 170337629481511
  }
  ```

**Fields**

- Request:

  | Field Name       | Type  | Description                                                                                                                        |
  | ---------------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------- |
  | video_duration   | `int` | Duration of the video in seconds. Its value must be greater than `0`.                                                              |
  | video_file_size  | `int` | Size of the video in MiB (1 MiB = 2^20 B).                                                                                         |
  | storage_duration | `int` | Duration in seconds for which you want to store the video stream in filecoin. Its value must be between `2628003` and `315360000`. |

- Response:

  | Field Name                 | Type  | Description                                                    |
  | -------------------------- | ----- | -------------------------------------------------------------- |
  | storage_cost_estimated     | `int` | Estimated storage cost in filecoin network (value in attoFIL). |
  | transcoding_cost_estimated | `int` | Estimated transcoding cost in livepeer network (value in WEI). |

<br>

## Requirements

- go 1.15.2
- ffmpeg
