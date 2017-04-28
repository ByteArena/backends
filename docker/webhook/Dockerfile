FROM almir/webhook

ENV APP_HOME /usr/src/app

COPY hooks.json /etc/webhook/hooks.json

RUN apk add --update-cache curl

CMD ["-verbose", "-hooks=/etc/webhook/hooks.json", "-hotreload"]
