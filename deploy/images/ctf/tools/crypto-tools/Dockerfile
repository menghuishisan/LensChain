FROM python:3.12-slim
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y curl wget git netcat-openbsd vim-tiny \
  && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir pycryptodome gmpy2 pwntools sympy
ENTRYPOINT ["/bin/sh"]
