# Rock Paper Scissors Game for Agones

A rock paper scissors game in Go deployed on K8s with Agones

## Requirements

- [mise](https://github.com/mise-rs/mise) for managing dependencies and running tasks
- Docker or Podman or any container runtime
- No firewall rules blocking the range 7000-8000

## Installation

```sh
export MISE_EXPERIMENTAL=true
mise run //...:init
mise run //...:dev
```

Once you're done:

```sh
mise run //...:cleanup
```
