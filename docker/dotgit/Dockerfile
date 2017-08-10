FROM golang:1.8-stretch

ARG DATABASE_URI
ARG GIT_REPOSITORIES_PATH
ARG MQ_HOST

ENV APP_HOME $GOPATH/src/github.com/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl netcat

RUN curl https://glide.sh/get | sh

RUN addgroup --gid 1000 git
RUN adduser --disabled-password git --uid 1000 --gid 1000

# Fixing environment (commands called by ssh are executed in another env space, with user git)
RUN touch /etc/dotgit.conf
RUN echo "{" >> /etc/dotgit.conf
RUN echo "\"DatabaseURI\":\"$DATABASE_URI\"," >> /etc/dotgit.conf
RUN echo "\"GitRepositoriesPath\":\"$GIT_REPOSITORIES_PATH\"," >> /etc/dotgit.conf
RUN echo "\"DockerHost\":\"$DOCKER_HOST\"," >> /etc/dotgit.conf
RUN echo "\"MqHost\":\"$MQ_HOST\"" >> /etc/dotgit.conf
RUN echo "}" >> /etc/dotgit.conf

COPY ./ $APP_HOME

WORKDIR $APP_HOME/bytearena/
RUN glide --debug install

# Build mq-cli
WORKDIR $APP_HOME/bytearena/cmd/mq-cli
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/mq-cli/mq-cli /usr/bin

# Build dotgit-ssh
WORKDIR $APP_HOME/bytearena/cmd/dotgit-ssh
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/dotgit-ssh/dotgit-ssh /usr/bin
RUN touch /var/log/dotgit-ssh.log && chown git:git /var/log/dotgit-ssh.log

# Build dotgit-hook-postreceive
WORKDIR $APP_HOME/bytearena/cmd/dotgit-hook-postreceive
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/dotgit-hook-postreceive/dotgit-hook-postreceive /home/git/git-scripts/post-receive
RUN chown -Rf git:git /home/git/git-scripts

# Build agentbuilder-cli
WORKDIR $APP_HOME/bytearena/cmd/agentbuilder-cli
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/agentbuilder-cli/agentbuilder-cli /usr/bin

# Build dotgit-keystore
WORKDIR $APP_HOME/bytearena/cmd/dotgit-keystore
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/dotgit-keystore/dotgit-keystore /usr/bin
RUN touch /var/log/dotgit-keystore.log && chown git:git /var/log/dotgit-keystore.log

# Build dotgit-mq-consumer
WORKDIR $APP_HOME/bytearena/cmd/dotgit-mq-consumer
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/dotgit-mq-consumer/dotgit-mq-consumer /usr/bin
RUN touch /var/log/dotgit-mq-consumer.log && chown git:git /var/log/dotgit-mq-consumer.log

# Setting up SSH
RUN mkdir -p /home/git/.ssh
RUN touch /home/git/.ssh/authorized_keys
RUN chmod 600 /home/git/.ssh/authorized_keys

#  Arguments to some keywords can make use of tokens, which are expanded at runtime:
#        %%    A literal `%'.
#        %f    The fingerprint of the key or certificate.
#        %h    The home directory of the user.
#        %t    The key or certificate type.
#        %u    The username.
#  AuthorizedKeysCommand accepts the tokens %%, %f, %h, %t, and %u.

RUN echo '' >> /etc/ssh/sshd_config
RUN echo 'AuthorizedKeysCommand /usr/bin/dotgit-keystore "%f"' >> /etc/ssh/sshd_config
RUN echo 'AuthorizedKeysCommandUser git' >> /etc/ssh/sshd_config

RUN echo "ChallengeResponseAuthentication no" >> /etc/ssh/sshd_config
RUN echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
RUN echo "UsePAM no" >> /etc/ssh/sshd_config

RUN echo "HostKey /opt/ssh/ssh_host_rsa_key"
RUN echo "HostKey /opt/ssh/ssh_host_dsa_key"
RUN echo "HostKey /opt/ssh/ssh_host_ecdsa_key"
RUN echo "HostKey /opt/ssh/ssh_host_ed25519_key"

COPY ./bytearena/docker/dotgit/template /home/git/template
COPY ./bytearena/docker/dotgit/git-scripts /home/git/git-scripts

# Permissions on /home/git
RUN mkdir -p /home/git/repositories
RUN chown -R git:git /home/git

# Startup script
COPY ./bytearena/docker/dotgit/start.sh $APP_HOME
WORKDIR $APP_HOME

# Installing docker cli (useful for agent-builder)

RUN apt-get --allow-unauthenticated -y install apt-transport-https ca-certificates curl software-properties-common
RUN apt-get -y install gnupg2
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"

RUN apt-get -y update
RUN apt-get -y install docker-ce

CMD ["bash", "start.sh"]
