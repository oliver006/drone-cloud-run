[![Build Status](https://cloud.drone.io/api/badges/oliver006/drone-cloud-run/status.svg)](https://cloud.drone.io/oliver006/drone-cloud-run) [![Coverage Status](https://coveralls.io/repos/github/oliver006/drone-cloud-run/badge.svg?branch=master)](https://coveralls.io/github/oliver006/drone-cloud-run?branch=master)

## drone-cloud-run - a Drone.io plugin to deploy images to Google Run







### Usage


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
      token:
        from_secret: google_credentials
      env_secret_api_key:
        from_secret: api_key_prod
```

