FROM perlt/workspace

USER root

RUN pacman -Sy
RUN pacman -S graphviz --noconfirm

USER perlt

RUN git config --global --add safe.directory /workspaces/BT-diagrams  
