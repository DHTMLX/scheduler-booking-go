FROM debian:bookworm-slim
WORKDIR /app
COPY ./scheduler-booking /app/scheduler-booking

EXPOSE 3000

CMD ["/app/scheduler-booking"]
