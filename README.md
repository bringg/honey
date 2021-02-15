# Honey

Honey is a simple tool for DevOps, let you simple search in few clouds for specific instance, written in pure Go.

## Motivation
I wanted a simple, fast all in one tool to search for instance by key in few clouds in parallel. Inspired by rclone, used few components from rclone tool.

## Backend providers

  * Amazon EC2
  * Consul by HashiCorp
  * Google Cloud Compute
  * Kubernetes Pods
  * MacStadium Mac Servers

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
curl -fsSL -o get_honey.sh https://raw.githubusercontent.com/bringg/honey/master/scripts/install.sh
chmod 700 get_honey.sh
./get_honey.sh
```

Yes, you can `curl https://raw.githubusercontent.com/bringg/honey/master/scripts/install.sh | bash` if you want to live on the edge.

 * Through Package Managers

From Homebrew (macOS)

```bash
brew tap bringg/honey
brew install honey
```

## Compile and run

```bash
go build .
./honey # --help to see all options
```

## Example

we can configure honey with os environment `HONEY_CONFIG_<backend name>_<options>`
to see more info about a particular backend use: `honey help backend <name>`

honey is support dot notation paths [GJSON Path Syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) and [JSONPath](https://kubernetes.io/docs/reference/kubectl/jsonpath/) same syntax as kubectl

```bash
# we can configure honey with env HONEY_CONFIG_<backend name>_<options>
export HONEY_CONFIG_GCP_PROJECTS=prj-1,prj-2,prj-3

# if k8s context is not set honey will try to use selected context from kube config, same as kubectl
export HONEY_CONFIG_K8S_CONTEXT=minikube

# if k8s namespace will be empty honey will try to search in all namespaces
export HONEY_CONFIG_K8S_NAMESPACE=default

# default honey output is table
honey -bk8s -f api
+--------------------------------------+--------------+-----------------------------+------+---------+-------------+----------------+
|                  ID                  | BACKEND NAME |            NAME             | TYPE | STATUS  | PRIVATE IP  |   PUBLIC IP    |
+--------------------------------------+--------------+-----------------------------+------+---------+-------------+----------------+
| e6a3aefa-fe55-45b3-a0b3-a45ef8263174 | k8s          | api-75f8ccd6bb-w4p9s        | pod  | Running | 172.17.0.18 | 192.168.64.129 |
| 23c0ad0f-5843-4254-953a-7a08bf84bc8d | k8s          | api-gateway-6b4c4ddf7-mfxd4 | pod  | Running | 172.17.0.14 | 192.168.64.129 |
+--------------------------------------+--------------+-----------------------------+------+---------+-------------+----------------+
```

json output with query
```bash
# default keys [id backend_name name type status private_ip public_ip]
# we can also query original backend instance object by specify `raw` key.
honey -bgcp -f test-instance -ojson=id,name,backend_name,raw.disks -vv
DEBU[0000] using cache: gcp, pattern `test-instance`, found: 4 items  operation=Find
DEBU[0000] using filter keys: [id name backend_name raw.disks]  where=place
[
  {
    "id": "3432432432",
    "name": "integration-test-instance-1111",
    "backend_name": "gcp",
    "raw.disks": [
      {
        "auto_delete": true,
        "guest_os_features": [
          {
            "force_send_fields": null,
            "null_fields": null,
            "type": "VIRTIO_SCSI_MULTIQUEUE"
          }
        ],
        "null_fields": null,
        "shielded_instance_initial_state": null,
        "source": "https://www.googleapis.com/compute/v1/projects/test-integration/zones/us-east1-d/disks/integration-rundeck-1111",
        "boot": true,
        "disk_size_gb": 100,
        "interface": "SCSI",
        "mode": "READ_WRITE",
        "type": "PERSISTENT",
        "disk_encryption_key": null,
        "initialize_params": null,
        "kind": "compute#attachedDisk",
        "licenses": [
          "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/licenses/ubuntu-1604-xenial"
        ],
        "device_name": "persistent-disk-0",
        "force_send_fields": null,
        "index": 0
      }
    ]
  }
]
```

```bash
honey -bgcp -f rundeck -ojson='name,backend_name,raw.disks.#(disk_size_gb>56).device_name' -vv

DEBU[0000] using cache: gcp, pattern `rundeck`, found: 4 items  operation=Find
DEBU[0000] using filter keys: [name backend_name raw.disks.#(disk_size_gb>56).device_name]  where=place
[
  {
    "name": "test-rundeck-93tw",
    "backend_name": "gcp",
    "raw.disks.#(disk_size_gb\u003e56).device_name": "persistent-disk-0"
  }
]
```

## Contribution

Feel free to open Pull-Request for small fixes and changes. For bigger changes and new backends please open an issue first to prevent double work and discuss relevant stuff.

License
-------

This is free software under the terms of MIT the license (check the
[LICENSE file](/LICENSE) included in this package).