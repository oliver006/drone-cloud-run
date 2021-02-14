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


## On Additional Flags

To be flexible with respect to flags that the `gcloud` command can accept, you
can use `addl_flags` in your drone setup settings. Use the flags are they're described
in the [documentation](https://cloud.google.com/sdk/gcloud/reference/run/deploy) without
the prefix `--` (eg. `--set-config-maps` becomes `set-config-maps`). If the flag doesn't
require any arguments, use `''` as the value.


