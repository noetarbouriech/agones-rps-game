# Rock Paper Scissors Demo Game for Agones

A rock paper scissors game and a matchmaking service made in Go deployed on Kubernetes with [Agones](https://agones.dev/).

![Matchmaking and Game demo](https://github.com/user-attachments/assets/03d1f8da-2ab8-4fdd-bd0a-46b4a3e0fa30)

## Overview

The repo contains two services:

- `matchmaking` — pairs players, allocates a game server via Agones, and redirect them to the game server address.
- `game` — a minimal game server that exposes a WebSocket endpoint for two players to play rock paper scissors.

This project makes extensive use of a tool called [mise](https://github.com/mise-rs/mise) which allowed me to manage my directories but also as a replacement for Makefiles/Taskfiles/etc. It is used to run [skaffold](https://skaffold.dev/) to work on the code but also to run [ko](https://ko.build/) in order to build container images and a lot more.

## Requirements

- [mise](https://github.com/mise-rs/mise) for managing dependencies and running tasks.
  - You must have `MISE_EXPERIMENTAL` set to true (`export MISE_EXPERIMENTAL=true`)
- Docker or Podman or any container runtime.
- No firewall rules blocking the range 7000-8000.

## Usage

First, you need to create the cluster:

```sh
mise run //...:init # inits the k8s cluster with kind and install agones
```

You can launch the prebuilt version of this project using:

```sh
mise run //...:run # applies the k8s manifests
```

You can dev on the project by running:

```sh
mise run //...:dev # runs skaffold for the matchmaking and game
```

You can do load-testing using k6 by running this:

```sh
# Make sure you first port-forward the matchmaking service as described below
mise run //...:test
```

Once you're done:

```sh
mise run //...:cleanup # deletes the kind cluster
```

## Accessing the matchmaking service

For now, you have to do a port-forward:

```sh
kubectl port-forward service/matchmaking 3000:80
```

Which makes the matchmaking service accessible on [localhost:3000](http://localhost:3000).
