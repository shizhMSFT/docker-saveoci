# docker-saveoci
A docker plugin to save one or more images to a tar archive in the OCI layout

## Install

Run the following command to install on Linux
```bash
mkdir ~/.docker/cli-plugins
curl -L https://github.com/shizhMSFT/docker-saveoci/releases/download/v0.1.0/docker-saveoci_0.1.0_linux_amd64.tar.gz | tar xvzC ~/.docker/cli-plugins/ docker-saveoci 
```

Help information can be reviewed by

```bash
docker help
```

## Save Images to an OCI-layout tarbal

The usage is exactly the same as `docker save`. Try

```bash
docker pull hello-world
docker saveoci -o hello.tar hello-world

# After saving, you may use other tools to process further.
oras manifest fetch --oci-layout hello.tar:latest
```
