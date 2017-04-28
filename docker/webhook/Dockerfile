FROM almir/webhook

ENV APP_HOME /usr/src/app

COPY hooks.json /etc/webhook/hooks.json

CMD ["-verbose", "-hooks=/etc/webhook/hooks.json", "-hotreload"]
