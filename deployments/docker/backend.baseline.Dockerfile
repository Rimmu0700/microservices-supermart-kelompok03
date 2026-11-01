FROM golang:latest
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o supermart-backend .
EXPOSE 3000 50051
CMD ["./supermart-backend"]