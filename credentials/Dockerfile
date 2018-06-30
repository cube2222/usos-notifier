FROM alpine:latest
RUN mkdir /app
RUN apk add --update ca-certificates
ADD cmd/cmd /app/
WORKDIR /app
EXPOSE 8080
EXPOSE 8081
EXPOSE 6666
CMD ["/app/cmd"]