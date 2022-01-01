FROM debian:jessie

RUN apt-get update -y && apt-get install -y ca-certificates

RUN mkdir -p /localfiles /s3cache

ENV GOROOT=/usr/local/go
ENV COMMUNITY_DB_DRIVER=sqlite3
ENV COMMUNITY_DB_URI=/community.db
ENV RELEASE_DB_DRIVER=sqlite3
ENV RELEASE_DB_URI=/release.db
ENV LOCALFILES_DB_DRIVER=sqlite3
ENV LOCALFILES_DB_URI=/localfiles.db
ENV LOCALFILES_STORAGE_ROOT=/localfiles
ENV KITE_S3CACHE=/s3cache

ADD user-node /user-node
ENTRYPOINT /user-node
