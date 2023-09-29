# crashloopbackon: delete CrashLoopBackoff pods

Small Go program to announce labeled CrashLoopBackoff pods to Datadog then delete them.

This lets them be recreated promptly, rather than delay a possible recovery.

Only affects containers labeled with `com.apollographql/crashloopbackon` = `true`.

Requires the `DD_API_KEY` environment variable, and Kubernetes authentication. Options in `-help`.

This process exits early on failures, and is expected to be restarted.

A Docker image is published to [DockerHub](https://hub.docker.com/repository/docker/pcarrier/crashloopbackon).
