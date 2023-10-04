# crashloopbackon: delete CrashLoopBackoff pods

Small Go program to announce CrashLoopBackoff pods labeled with `crashloopbackon` = `true` to Slack then delete them.

This lets them be recreated promptly, rather than delay a possible recovery.

Requires the `SLACK_TOKEN` environment variable, and Kubernetes authentication. Options in `-help`.

This process exits early on failures, and is expected to be restarted.

A Docker image is published to [DockerHub](https://hub.docker.com/repository/docker/pcarrier/crashloopbackon).
