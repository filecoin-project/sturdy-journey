# Sturdy Journey

Sturdy journey is a generic event / api handler with the goal of providing a simple way to connect automation between
projects in the filecoin project organization. Uses can range from helping to keep senseitive secrets out of public CircleCI
projects to redirect webhook events from any services, but most notably Github webhook events.

## Install

A kubernetes resource file is provided with our primarly deployment configured. In addtion to install the file, the secrets
required for using the sturdy journey need to be installed manually. Requirements for the secrets are document in the resource
file.

See `./manifests/sturdy-journey.yaml` for some additional information.

```
kubectl create namespace sturdy-journey
kubectl -n sturdy-journey apply -f ./manifests/sturdy-journey.yaml
```

## Contributing

PRs accepted.

## License

Dual-licensed under [MIT](https://github.com/filecoin-project/sturdy-journey/blob/master/LICENSE-MIT) + [Apache 2.0](https://github.com/filecoin-project/sturdy-journey/blob/master/LICENSE-APACHE)
