# Honey

Honey is a simple tool for DevOps, let you simple search in few clouds for specific instance, written in pure Go.

## Motivation
I wanted a simple, fast all in one tool to search for instance by key in few clouds in parallel. Inspired by rclone, used few components from rclone tool.

## Backend providers

  * Amazon EC2
  * Google Cloud Compute
  * Kubernetes Pods

## Installing Honey
* From the Binary Releases

Every release of Honey provides binary releases for a variety of OSes. These binary versions can be manually downloaded and installed.

Download your desired version
Unpack it (unzip honey_Darwin_x86_64.zip)
Find the honey binary in the unpacked directory, and move it to its desired destination (mv honey /usr/local/bin/honey)
From there, you should be able to run the honey.

*  From Script

Honey now has an installer script that will automatically grab the latest version of Honey and install it locally.

You can fetch that script, and then execute it locally. It's well documented so that you can read through it and understand what it is doing before you run it.

```bash
curl -fsSL -o get_honey.sh https://raw.githubusercontent.com/shareed2k/honey/master/scripts/install.sh
chmod 700 get_honey.sh
./get_honey.sh
```

Yes, you can `curl https://raw.githubusercontent.com/shareed2k/honey/master/scripts/install.sh | bash` if you want to live on the edge.

 * Through Package Managers

From Homebrew (macOS)

```bash
brew tap shareed2k/honey
brew install honey
```

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