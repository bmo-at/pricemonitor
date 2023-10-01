FROM golang:latest
RUN mkdir /google-chrome/
RUN apt update && wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb && apt install -y ./google-chrome-stable_current_amd64.deb
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o main .
 
CMD ["/app/main"]
