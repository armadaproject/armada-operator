# Submit Job

`armadactl` is a command line tool which is used to interact with the Armada API.
You can find a list of supported platforms and architectures in the Armada [Releases](https://github.com/armadaproject/armada/releases).
Let's run the following command to install the `armadactl` CLI:
```bash
curl -L -o /tmp/armadactl "https://github.com/armadaproject/armada/releases/download/v0.3.101/armadactl_0.3.101_darwin_all.tar.gz" && \
  tar -xzvf /tmp/armadactl              && \
  mv armadactl /usr/local/bin/armadactl && \
  rm /tmp/armadactl                     && \
  chmod +x /usr/local/bin/armadactl
```
