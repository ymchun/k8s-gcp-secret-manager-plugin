# k8s-gcp-secret-manager-plugin

[![Build Status](https://travis-ci.org/ymchun/k8s-gcp-secret-manager-plugin.svg?branch=master)](https://travis-ci.org/ymchun/k8s-gcp-secret-manager-plugin)

KMS provider for Kubernetes encryption configuration using GCP Secret Manager

## Prerequisite

- You have created a secret and store it into GCP Secret Manager.
- You have a JSON format credentials file of a service account with proper role permission.
  - Required Roles:
    - `Secret Manager Secret Accessor`
    - `Secret Manager Viewer`

## Install as a Linux service

```
[Unit]
Description=K8S GCP Secret Manager Plugin
After=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=on-failure
RestartSec=1
User=root
ExecStart=/bin/k8splugin -credentials path/to/gcp/credentials/file -key-uri projects/project-id/secrets/my-key-name/versions/1
SyslogIdentifier=k8splugin

[Install]
WantedBy=multi-user.target
```
1. Create a new file called `k8splugin.service` with the following content:
2. Put the file to path `/etc/systemd/system/k8splugin.service`
3. Start the service `systemctl start k8splugin`
4. Start the service on boot `systemctl enable k8splugin`

## Command line arguments

Example command:
```
$ k8splugin -credentials path/to/gcp/credentials/file -key-uri projects/project-id/secrets/my-key-name/versions/1
```

Arguments:
```
-alsologtostderr
    log to standard error as well as files

-credentials string
    Path to GCP Secret Manager service account credentials JSON file

-key-uri projects/*/secrets/*/versions/*
    Resource ID of the secret in the format projects/*/secrets/*/versions/*

-log_backtrace_at value
    when logging hits line file:N, emit a stack trace

-log_dir string
    If non-empty, write log files in this directory

-logtostderr
    log to standard error instead of files

-stderrthreshold value
    logs at or above this threshold go to stderr

-unix-socket string
    Full path to Unix socket that is used for communicating with KubeAPI Server,
    or Linux socket namespace object - must start with @ (default "/var/run/k8splugin.sock")

-v value
    log level for V logs

-vmodule value
    comma-separated list of pattern=N settings for file-filtered logging
```

## Configuration (For more detail, see [Learn more](#learn-more) section)

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
      - secrets
    providers:
      - kms:
          name: GCPSecretManager
          # this should match your k8splugin socket path
          # set with -unix-socket or default: /var/run/k8splugin.sock
          endpoint: unix:///var/run/k8splugin.sock
          cachesize: 10
          timeout: 3s
      - identity: {}
```
1. Create a new encryption configuration file using the appropriate properties for the kms provider:
2. Set the `--encryption-provider-config=path-to-yaml-file` flag on the kube-apiserver to point to the location of the configuration file.
3. Restart your API server.

## Learn more

- Read [GCP Secret Manager documentation](https://cloud.google.com/secret-manager/docs)
- Read [Using a KMS provider for data encryption](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/)
