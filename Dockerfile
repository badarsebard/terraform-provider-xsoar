FROM ubuntu

ARG installer_path
ARG installer
ARG multitenant=""
ARG elastic_url=""
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update
RUN apt-get install -y docker.io
COPY "${installer_path}/${installer}" ./installer.sh
RUN "./installer.sh -- -y -do-not-start-server ${multitenant} ${elastic_url}"
RUN rm ${installer}

CMD ["/usr/local/demisto/server"]
