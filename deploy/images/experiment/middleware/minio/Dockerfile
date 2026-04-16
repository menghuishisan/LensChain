FROM minio/minio:RELEASE.2025-02-28T09-55-16Z
EXPOSE 9000 9001
ENTRYPOINT ["minio", "server", "/data", "--console-address", ":9001"]
