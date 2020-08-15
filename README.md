# Demux

A gateway to facilitate a decentralised streaming ecosystem.

## Getting Started

- Clone the repo and update submodules:
  - `git clone https://github.com/buidl-labs/Demux`
  - `git submodule update --init --recursive`
- Running the filecoin localnet using powergate:
  - Start your docker daemon
  - `cd Demux/powergate/docker`
  - Start the localnet: `make localnet`
- Download the latest build of livepeer pull mode:
  - linux: https://build.livepeer.live/0.5.8-aed26dfc/livepeer-linux-amd64.tar.gz
  - mac: https://build.livepeer.live/0.5.8-aed26dfc/livepeer-darwin-amd64.tar.gz
- Place the `livepeer` binary inside `Demux/livepeerPull/` directory.
- In a new terminal window, set environment variables (sample present in `Demux/.env.sample`)
- Run server: `make run`

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
    "CID": "bafybeifpkabn6hkh4njamtiwcj45p4ig3eibfy6ac2jhltvssyf4eoes54",
    "Expiry": 1000000,
    "Miner": "t01000",
    "RootCID": "QmdhwHpiZM53nRCVhftxk2aw8qFvMdw2AZV2ABi74t13ad",
    "StorageCost": 1000000000,
    "TranscodingCost": "2.838960491e+13"
  }
  ```

## Requirements

- go 1.14.4
- ffmpeg
