FROM golang:1.8-stretch

ARG GIT_ADMIN_KEY_PUBLIC
ARG DATABASE_URI
ARG GIT_REPOSITORIES_PATH
ARG API_PORT

ENV BA_ROOT $GOPATH/src/github.com/bytearena
ENV APP_HOME $BA_ROOT/dotgit

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl

RUN adduser --disabled-password git

COPY ./dotgit $APP_HOME

# Fixing environment (commands called by ssh are executed in another env space, with user git)
RUN touch /etc/dotgit.conf
RUN echo "{" >> /etc/dotgit.conf
RUN echo "\"DatabaseURI\":\"$DATABASE_URI\"," >> /etc/dotgit.conf
RUN echo "\"GitRepositoriesPath\":\"$GIT_REPOSITORIES_PATH\"," >> /etc/dotgit.conf
RUN echo "\"ApiPort\":$API_PORT" >> /etc/dotgit.conf
RUN echo "}" >> /etc/dotgit.conf

# Build dotgit-ssh
WORKDIR $APP_HOME/cmd/dotgit-ssh
RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/cmd/dotgit-ssh/dotgit-ssh /usr/bin

# Build dotgit-api
WORKDIR $APP_HOME/cmd/dotgit-api
RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/cmd/dotgit-api/dotgit-api /usr/bin

# Build dotgit-keystore
WORKDIR $APP_HOME/cmd/dotgit-keystore
RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/cmd/dotgit-keystore/dotgit-keystore /usr/bin
RUN touch /var/log/dotgit-keystore.log && chown git:git /var/log/dotgit-keystore.log

# Setting up SSH
RUN mkdir -p /home/git/.ssh
RUN touch /home/git/.ssh/authorized_keys
RUN chmod 600 /home/git/.ssh/authorized_keys

#RUN echo -n 'no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty,command="/usr/bin/dotgit-ssh _agentbuilder"' >> /home/git/.ssh/authorized_keys
#RUN echo -n ' ' >> /home/git/.ssh/authorized_keys
#RUN echo "$GIT_ADMIN_KEY_PUBLIC" | base64 -d >> /home/git/.ssh/authorized_keys

#  Arguments to some keywords can make use of tokens, which are expanded at runtime:
#        %%    A literal `%'.
#        %f    The fingerprint of the key or certificate.
#        %h    The home directory of the user.
#        %t    The key or certificate type.
#        %u    The username.
#  AuthorizedKeysCommand accepts the tokens %%, %f, %h, %t, and %u.

RUN echo '' >> /etc/ssh/sshd_config
RUN echo 'AuthorizedKeysCommand /usr/bin/dotgit-keystore "%t" "%f"' >> /etc/ssh/sshd_config
RUN echo 'AuthorizedKeysCommandUser git' >> /etc/ssh/sshd_config

RUN echo "ChallengeResponseAuthentication no" >> /etc/ssh/sshd_config
RUN echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
RUN echo "UsePAM no" >> /etc/ssh/sshd_config

# Permissions on /home/git
RUN mkdir -p /home/git/repositories
RUN chown -R git:git /home/git

# Startup script
COPY ./bytearena/docker/dotgit/start.sh $APP_HOME
WORKDIR $APP_HOME

CMD ["bash", "start.sh"]