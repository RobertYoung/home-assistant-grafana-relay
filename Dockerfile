FROM scratch

COPY home-assistant-grafana-relay /usr/bin/

ENTRYPOINT ["/usr/bin/home-assistant-grafana-relay"]