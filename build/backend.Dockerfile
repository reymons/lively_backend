FROM golang:alpine AS base
WORKDIR /app
RUN apk update && apk add make

FROM base AS build
COPY . .
RUN make build

FROM base AS final
COPY --from=build /app/bin/backend .
CMD ["./backend"]
