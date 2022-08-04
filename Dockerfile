FROM ubuntu

RUN apt-get update && apt-get install -y fuse3

COPY dummy-fuse /bin/dummy-fuse
COPY dummy-fuse-csi /bin/dummy-fuse-csi
COPY dummy-fuse-workload /bin/dummy-fuse-workload
