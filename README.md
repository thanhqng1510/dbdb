# Minimal distributed key-value store using Raft

## Building

To build the `dbdb` application, ensure you have Go installed on your system. Then, navigate to the project's root directory and run:

```bash
$ make build
```

This will compile the application and place the executable at `./bin/dbdb`.

## Running

After building, you can run `dbdb`. The following flags are **required**:
* `--node-id <id>`: A unique identifier for this node (e.g., `node1`).
* `--raft-port <port>`: The port for Raft internode communication (e.g., `2222`).
* `--http-port <port>`: The port for the HTTP API (e.g., `8222`).

Optional flags:
*   `--bootstrap`: Use this flag for the *first* node when starting a new cluster. Do not use with `--join`.
*   `--join <leader-http-address>`: The HTTP address of an existing leader node to join (e.g., `localhost:8222`). Do not use with `--bootstrap`.

## Running a Multi-Node Cluster with Docker Compose

A sample `docker-compose.yml` is provided. To start a 3-node cluster:

```bash
docker-compose up
```

This will start three nodes (`node1` as leader, `node2` and `node3` as follower)

**Note:** When using Docker Compose, service names (e.g., `node1:8221`) can be used for inter-node communication.

## Example: Manually starting a 2-node cluster

Terminal 1: Start the first node (bootstrap)

```bash
$ ./bin/dbdb --node-id node1 --raft-port 2221 --http-port 8221 --bootstrap
```

Terminal 2: Start the second node and join the first node

```bash
$ ./bin/dbdb --node-id node2 --raft-port 2222 --http-port 8222 --join localhost:8221
```

Terminal 3, now add a key:

```bash
$ curl -X POST 'localhost:8221/apply' -d '{"op": "set", "key": "x", "value": "23"}' -H 'content-type: application/json'
```

Terminal 3, now get the key from either server:

```bash
$ curl 'localhost:8221/get?key=x'
{"data":"23"}
$ curl 'localhost:8222/get?key=x'
{"data":"23"}
```

Terminal 3, now delete key 'x'

```bash
$ curl -X POST 'localhost:8221/apply' -d '{"op": "del", "key": "x"}' -H 'content-type: application/json'
```

Terminal 3, now get the key from either server:

```bash
$ curl 'localhost:8221/get?key=x'
{"data":""}
$ curl 'localhost:8222/get?key=x'
{"data":""}
```

References:

* https://yusufs.medium.com/creating-distributed-kv-database-by-implementing-raft-consensus-using-golang-d0884eef2e28
* https://github.com/Jille/raft-grpc-example
* https://github.com/otoolep/hraftd
* https://pkg.go.dev/github.com/hashicorp/raft
* https://www.youtube.com/watch?v=8XbxQ1Epi5w