> [Concourse](https://concourse-ci.org) is an open-source continuous thing-doer.

At Kite, we use Concourse for at least parts of our build/deploy pipelines.
The goal is to incrementally port all deployment jobs to Concourse,
but in the meanwhile, our prior build system (Solness) will trigger Concourse jobs as needed.

In order to manually run the pipelines, you can login at [concourse.kite.com](http://concourse.kite.com)
from within the AWS dev VPN. Find credentials in Quip.

For now we *do not* intend to move developer CI onto to Concourse,
since scaling up a self-hosted CI system comes with its own set of challenges,
and our current solution (Travis) is "good enough." This is purely for deployments.

## Development

Read the Concourse docs!

Pipelines are composed of jobs which are in turn composed of tasks.
We have a pipeline called "release" defined in `pipelines/release/pipeline.ytt`.

In order to develop this pipeline, you need the Concourse `fly` tool,
as well as the YAML templating tool `ytt`.

This pipeline can be updated using the `fly` CLI tool, or with the `make` command:
```
make pipelines/deploy/set
```

### Secrets

All secrets are currently stored in AWS Systems Manager Parameter Store in us-west-1.
The Concourse Web node is configured to look up secrets from SSM.

## Provisioning a Worker

Eventually, we should use Packer to provision worker AMIs, but for now
workers must be manually configured.

### Windows

1. Start with a "Windows Server 2019 with Containers" machine image.
2. Provision all the tools needed for building Kite,
   as per the Windows [README](../windows/README.md).
    * also `choco install windows-sdk-10.0`
3. Allocate and mount a separate disk (100G) for all Concourse-related data
    * Below, we assume it's mounted at `D:`.
    * `mkdir D:\containers`, `mkdir D:\concourse`
4. Enable long paths using registry editor.
    * set `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\FileSystem\LongPathsEnabled` to 1.
5. Download [WinSW](https://github.com/kohsuke/winsw)
   and [Concourse](https://github.com/concourse/concourse/).
    * Move Concourse binary to `D:\concourse\concourse-bin.exe`
    * Move the WinSW binary to `D:\concourse\concourse.exe`
5. Provision a worker key on the Windows machine.
```
cd D:\concourse
ssh-keygen -t rsa -b 4096 -f tsa-worker-key
...
cat D:\concourse\tsa-worker-key.pub
```
    * add the public key to `authorized_keys` on the Concourse web node.
    * restart the web node.
6. Create `D:\concourse\concourse.xml` to configure all the Concourse options.
```
<service>
  <id>concourse</id>
  <name>Concourse</name>
  <description>Concourse Windows worker.</description>
  <startmode>Automatic</startmode>
  <executable>D:\concourse\concourse-bin.exe</executable>
  <argument>worker</argument>
  <argument>/work-dir</argument>
  <argument>D:\containers</argument>
  <argument>/tsa-worker-private-key</argument>
  <argument>D:\concourse\tsa-worker-key</argument>
  <argument>/tsa-public-key</argument>
  <argument>D:\concourse\tsa-host-key.pub</argument>
  <argument>/tsa-host</argument> <argument>10.86.0.122:2222</argument>
  <onfailure action="restart" delay="10 sec"/>
  <onfailure action="restart" delay="20 sec"/>
  <logmode>rotate</logmode>
</service>
```
7. Install and start the Concourse service
```
D:\concourse\concourse.exe install
D:\concourse\concourse.exe start
```
8. License VS Community 2019 under the system user.
    * Download [`PsExec.exe`](https://docs.microsoft.com/en-us/sysinternals/downloads/psexec)
    * Start VS under the system user: `PsExec.exe -sid "C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\IDE\devenv.com"`
    * Log in to license the software.
