# agglayer

agglayer is a web service that receives ZKPs from different CDK chains and checks the soundness of them before sending the ZKP to L1 for verification.

To find out more about Polygon, visit the [official website](https://wiki.polygon.technology/docs/cdk/).

WARNING: This is a work in progress so architectural changes may happen in the future. The code is still being audited, so please contact the Polygon team if you would like to use it in production.

## Getting Started

### Prerequisites

This is an example of how to list things you need to use the software and how to install them.
* docker
* docker compose

## Usage

### Running in local with Docker

Run
```
make run-docker
```

## Production setup

Currently only one instance of agglayer can be running at the same time, so it should be automatically started in the case of failure using a containerized setup or an OS level service manager/monitoring system.

### Installation

1. Clone the repo
   ```sh
   git clone https://github.com/0xPolygon/agglayer.git
   ```
3. Install Golang dependencies
   ```sh
   go install .
   ```

### Prerequisites

* For each CDK chain it's necessary to configure it's corresponding RPC node, synced with the target CDK, this node is for checking the state root after executions of L2 batches.
* It's recommended to have a durable HA PostgresDB for storage, prefer AWS Aurora Postgres or Cloud SQL for postgres in GCP.

### Configuration of `agglayer.toml`
    * Configure `[FullNodeRPCs]` to point to the corresponding L2 full node.
    * Configure `[L1]` to point to the corresponding L1 chain.
    * Configure the `[DB]` section with the managed database details.

## Copyright 2022 Polygon Technology

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
