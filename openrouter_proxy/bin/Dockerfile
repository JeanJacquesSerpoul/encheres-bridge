FROM alpine:3.19

# ca-certificates: required for TLS calls
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY openrouter_proxy ./openrouter_proxy
RUN chmod +x openrouter_proxy

EXPOSE 9013


CMD ["./openrouter_proxy"]
