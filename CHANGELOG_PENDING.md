### Improvements

* Add an `ecs` deploy target that runs each job as an Amazon ECS task, on Fargate by default
* Keep a job alive while its image is pulled, so a slow pull of a large image is no longer reaped as a stalled job

### Bug Fixes

* Retry an image pull whose registry stream is interrupted, instead of hanging
* Make job files readable by jobs whose image runs as a non-root user
* Report the agent's version in tarball installs. Previously only the container image knew its version
* Fix `vpcId` config key typo in `agent-images/agent-setup/` that prevented the optional VPC override from working
