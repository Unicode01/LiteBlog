FROM alpine:latest AS builder

COPY release/LiteBlog_linux_amd64.zip /tmp/
RUN mkdir /liteblog && \
    apk add unzip && \
    unzip /tmp/LiteBlog_linux_amd64.zip -d /liteblog && \
    rm /tmp/LiteBlog_linux_amd64.zip

FROM scratch

COPY --from=builder /liteblog /liteblog

WORKDIR /liteblog
CMD ["/liteblog/LiteBlog"]