FROM golang:1.12-stretch AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .
RUN go build main.go

FROM ubuntu:18.04
ENV DEBIAN_FRONTEND=noninteractive
ENV PGVER 11
EXPOSE 5000

RUN apt-get update && apt-get install -y wget gnupg && \
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
RUN echo "deb http://apt.postgresql.org/pub/repos/apt bionic-pgdg main" > /etc/apt/sources.list.d/PostgreSQL.list

RUN apt-get update && apt-get install -y postgresql-$PGVER


WORKDIR TP_DB_RK2
COPY . .

# Подключаемся к PostgreSQL и создаем БД
USER postgres
RUN /etc/init.d/postgresql start &&\
    psql --command "CREATE USER docker WITH SUPERUSER PASSWORD 'docker';" &&\
    createdb -O docker docker &&\
    psql -d docker -c "CREATE EXTENSION IF NOT EXISTS citext;" &&\
    psql docker -a -f sql/init.sql &&\
    /etc/init.d/postgresql stop

USER root
# Настраиваем сеть и БД
RUN echo "local all all md5" > /etc/postgresql/$PGVER/main/pg_hba.conf && \
    echo "host all all 0.0.0.0/0 md5" >> /etc/postgresql/$PGVER/main/pg_hba.conf
RUN cat sql/postgresql.conf >> /etc/postgresql/$PGVER/main/postgresql.conf
VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]
EXPOSE 5432

EXPOSE 5000

COPY --from=builder /app/main .
# Запускаем PostgreSQL и api сервер
CMD service postgresql start && ./main
