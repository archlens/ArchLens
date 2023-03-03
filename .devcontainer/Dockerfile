FROM perlt/workspace

ENV SHELL /bin/bash

RUN sudo apk add curl
RUN sudo apk add python3=3.9.16-r0 --repository=https://dl-cdn.alpinelinux.org/alpine/v3.15/main
RUN sudo apk add python3-dev=3.9.16-r0 --repository=https://dl-cdn.alpinelinux.org/alpine/v3.15/main

RUN curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py
RUN python3.9 get-pip.py

RUN echo 'export PATH="$PATH:/home/perlt/.local/bin"' >> ~/.bashrc
RUN git config --global --add safe.directory /workspaces/MT-diagrams
