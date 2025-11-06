FROM postgres:16-alpine

# Copy SSL certificates
COPY certs/server.crt /var/lib/postgresql/server.crt
COPY certs/server.key /var/lib/postgresql/server.key

# Set ownership and permissions  
RUN chown postgres:postgres /var/lib/postgresql/server.crt /var/lib/postgresql/server.key && \
    chmod 600 /var/lib/postgresql/server.key && \
    chmod 644 /var/lib/postgresql/server.crt
