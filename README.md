[![Build Status](https://cloud.drone.io/api/badges/oliver006/drone-cloud-run/status.svg)](https://cloud.drone.io/oliver006/drone-cloud-run) [![Coverage Status](https://coveralls.io/repos/github/oliver006/drone-cloud-run/badge.svg)](https://coveralls.io/github/oliver006/drone-cloud-run)

## drone-cloud-run - A drone.io plugin to deploy Google Cloud Run containers. 


### Usage

For usage in drone v1.0:
```
kind: pipeline
name: default

steps:
  - name: deploy-using-new-drone-plugin-version
    image: oliver006/drone-cloud-run:latest
    pull: always
    settings:
      action: deploy                                            # other actions: update-traffic
      service: my-api-service
      runtime: gke                                              # default=managed
      image: org-name/my-api-service-image
      timeout: 10m                                              # google cloud default is 5m
      memory: 512Mi
      variant: alpha                                            # uses "gcloud alpha run" command variant, default=<empty string>. Other supported variant is beta.
      region: us-central1
      allow_unauthenticated: true                               # default=false
      svc_account: 1234-my-svc-account@google.svcaccount.com 
      addl_flags:                                               # if present, flags passed to command
        add-cloud-sql-instances: instance1,instance2
      token:
        from_secret: google_credentials
      environment:
        VAR_1: "var01"
        ANOTHER_ENV_VAR: "env_var_value"
      secrets:                                                  # set environment variables or file path contents to a Secret Manager secret value
        MY_SECRET: "my-secret:latest"
        /mount/path: "mounted-secret:latest"
      env_secret_api_key:
        from_secret: api_key_prod
```

For usage in drone v0.8:
```
kind: pipeline
name: default

steps:
  - name: deploy-using-new-drone-plugin-version
    image: oliver006/drone-cloud-run:latest
    pull: always

    # plugin settings are top-level
    action: deploy
    service: my-api-service
    deployment_image: org-name/my-api-service-image
    memory: 512Mi
    region: us-central1
    allow_unauthenticated: true
    svc_account: 1234-my-svc-account@google.svcaccount.com
    addl_flags:
        clear-cloudsql-instances: ''
    secrets:
      - source: google_credentials
        target: token
      - source: api_key_prod
        target: env_secret_api_key

```

### Updating traffic

You can optionally use the `update-traffic` action to change which revisions
will receive traffic by passing the [`services update-traffic`](https://cloud.google.com/sdk/gcloud/reference/alpha/run/services/update-traffic)
command's `--to-latest`, `--to-revisions` or `--to-tags` arguments to `addl_flags`.

See the [Cloud Run traffic routing](https://cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration)
docs for more information.

```
kind: pipeline
name: default

steps:
  - name: deploy-using-new-drone-plugin-version                 # Same options available as above
    image: oliver006/drone-cloud-run:latest
    settings:
      action: deploy
      variant: alpha
      service: my-api-service
      runtime: gke
      image: org-name/my-api-service-image
      region: us-central1
      addl_flags:
        no-traffic: ''                                          # Don't route 100% of traffic to this revision by default
        tag: canary                                             # Human-readable revision name
      token:
        from_secret: google_credentials

  - name: update-traffic-using-new-drone-plugin-version
    image: oliver006/drone-cloud-run:latest
    settings:
      action: update-traffic
      variant: alpha
      service: my-api-service
      runtime: gke
      region: us-central1
      addl_flags:
        to-tags: canary=5                                       # One of: to-latest, to-revisions, to-tags (alpha variant only)
      token:
        from_secret: google_credentials
```

## On Additional Flags

To be flexible with respect to flags that the `gcloud` command can accept, you
can use `addl_flags` in your drone setup settings. Use the flags are they're described
in the [documentation](https://cloud.google.com/sdk/gcloud/reference/run/deploy) without
the prefix `--` (eg. `--set-config-maps` becomes `set-config-maps`). If the flag doesn't
require any arguments, use `''` as the value.


