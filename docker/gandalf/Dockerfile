FROM ubuntu:latest

ARG GIT_ADMIN_KEY_PUBLIC

ENV DEBIAN_FRONTEND "noninterractive"

ENV APP_HOME /usr/src/app

RUN mkdir $APP_HOME
WORKDIR $APP_HOME

RUN apt-get update -y
RUN apt-get install software-properties-common -y

RUN apt-add-repository ppa:tsuru/ppa
RUN apt-get update -y

RUN apt-get install rsyslog gandalf-server upstart openssh-server curl -y

COPY ./docker/gandalf/ $APP_HOME
COPY ./docker/gandalf/template /home/git/template
RUN chmod +x start.sh

RUN mkdir -p /home/git/.ssh
RUN touch /home/git/.ssh/authorized_keys
RUN chmod 600 /home/git/.ssh/authorized_keys

RUN mkdir -p /home/git/repositories
RUN chown -R git:git /home/git

COPY ./docker/gandalf/gandalf.conf /etc/gandalf.conf

RUN echo "$GIT_ADMIN_KEY_PUBLIC" | base64 -d >> /home/git/.ssh/authorized_keys

CMD ["bash", "start.sh"]
