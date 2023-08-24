FROM rabbitmq:3.12.3-management

COPY ./static /plugins

RUN rabbitmq-plugins enable rabbitmq_delayed_message_exchange