FROM golang:1.8-stretch

ARG GIT_ADMIN_KEY_PUBLIC

ENV BA_ROOT $GOPATH/src/github.com/bytearena
ENV APP_HOME $BA_ROOT/dotgit

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl

RUN adduser --disabled-password git

COPY ./ $APP_HOME
WORKDIR $APP_HOME/cmd/dotgit-ssh

RUN go get -v ./...
RUN go build

RUN ls -lah $APP_HOME/cmd/dotgit-ssh/

RUN cp -vf $APP_HOME/cmd/dotgit-ssh/dotgit-ssh /usr/bin

RUN mkdir -p /home/git/.ssh
RUN touch /home/git/.ssh/authorized_keys
RUN chmod 600 /home/git/.ssh/authorized_keys

RUN mkdir -p /home/git/repositories
RUN chown -R git:git /home/git

RUN echo -n 'no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty,command="/usr/bin/dotgit-ssh _agentbuilder"' >> /home/git/.ssh/authorized_keys
RUN echo -n ' ' >> /home/git/.ssh/authorized_keys
RUN echo "$GIT_ADMIN_KEY_PUBLIC" | base64 -d >> /home/git/.ssh/authorized_keys

WORKDIR $BA_ROOT/bytearena/docker/dotgit

RUN pwd

RUN ls -lah --color

CMD ["bash", "start.sh"]
#CMD ["su", "git", "-c", "/usr/bin/git", "daemon", "--verbose", "--base-path=/home/git/repositories", "--export-all"]