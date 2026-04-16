FROM eclipse-temurin:21-jdk
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y git curl maven gradle \
  && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/bin/bash"]
