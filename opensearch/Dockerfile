FROM opensearchproject/opensearch:2.11.0

USER root

# Copy scripts
COPY scripts/setup-security.sh /usr/local/bin/setup-security.sh
COPY scripts/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod +x /usr/local/bin/setup-security.sh /usr/local/bin/entrypoint.sh

USER opensearch

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
