FROM golang:1.18.1-alpine AS build

COPY . /opt/earthwalker/

WORKDIR /opt/earthwalker

RUN apk update && apk add --no-cache make npm && make

FROM alpine

WORKDIR /opt/earthwalker

# Copy only built assets from build image
COPY --from=build /opt/earthwalker/earthwalker .
COPY --from=build /opt/earthwalker/public/ ./public/

ENTRYPOINT ["./earthwalker"]