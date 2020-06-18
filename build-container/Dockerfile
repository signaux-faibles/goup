FROM alpine
ARG binary
COPY ./$binary /app/goup
RUN chmod 555 /app/goup
WORKDIR /app
EXPOSE 3000
CMD ["/app/goup"]