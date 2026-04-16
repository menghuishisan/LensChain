FROM ubuntu:24.04
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y gdb binutils radare2 file python3 python3-pip strace ltrace \
  && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir pwntools
ENTRYPOINT ["/bin/bash"]
