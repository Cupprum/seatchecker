FROM docker.io/library/fedora:39

RUN dnf update -y && dnf install -y \
    git \
    golang \
    docker

RUN cd /usr/local; curl -L https://dl.dagger.io/dagger/install.sh | sh