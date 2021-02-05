# Honey

Honey is a simple tool for DevOps, let you simple search in few clouds for specific instance, written in pure Go.

## Motivation
I wanted a simple, fast all in one tool to search for instance by key in few clouds in parallel. Inspired by rclone, used few components from rclone tool.

## Backend providers

  * Amazon EC2
  * Google Cloud Compute
  * Kubernetes Pods

## Compile and run

```bash
go build .
./honey # --help to see all options
```

## Contribution

Feel free to open Pull-Request for small fixes and changes. For bigger changes and new backends please open an issue first to prevent double work and discuss relevant stuff.

License
-------

This is free software under the terms of MIT the license (check the
[LICENSE file](/LICENSE) included in this package).