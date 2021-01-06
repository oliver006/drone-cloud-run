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
      action: deploy
      service: my-api-service
      image: org-name/my-api-service-image
      memory: 512Mi
      region: us-central1
      allow_unauthenticated: true
      cloud_sql_instances: +instance1,instance2
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
    secrets:
      - source: google_credentials
        target: token
      - source: api_key_prod
        target: env_secret_api_key

```

### On CloudSqlInstances

For the `gcloud` flags related to [modifying cloud sql instances](https://cloud.google.com/sdk/gcloud/reference/run/deploy#--add-cloudsql-instances) connected to the service,

```

--add-cloudsql-instances=[CLOUDSQL-INSTANCES,…]
    Append the given values to the current Cloud SQL instances. 
--clear-cloudsql-instances
    Empty the current Cloud SQL instances. 
--remove-cloudsql-instances=[CLOUDSQL-INSTANCES,…]
    Remove the given values from the current Cloud SQL instances. 
--set-cloudsql-instances=[CLOUDSQL-INSTANCES,…]
    Completely replace the current Cloud SQL instances with the given values. 
```

we provide a singular field `cloud_sql_update` since you can only set at most one.

To use it, here are some examples:

| arg                             | written as |
|---------------------------------|------------|
| --add-cloudsql-instances=a,b,e  | +abc       |
| --clear-cloudsql-instances      | #          |
| --remove-cloudsql-instances=a,c | -a,c       |
| --set-cloudsql-instances=a,d,c  | =a,d,c     |

