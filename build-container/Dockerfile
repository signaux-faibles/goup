FROM alpine
ARG path
COPY ["./$path/goup", "./$path/linkFile.sh", "entrypoint.sh", "/app/"]
RUN apk add --no-cache clamav sudo;\
    chmod 555 /app/goup;\
    chmod 555 /app/entrypoint.sh;\
    chmod 555 /app/linkFile.sh;
WORKDIR /app
EXPOSE 3000
ENTRYPOINT ["/app/entrypoint.sh"]