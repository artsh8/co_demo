FROM python:3.12-slim

WORKDIR /dummy_data
COPY ./dummy_data/requirements.txt requirements.txt
RUN pip install -r requirements.txt
COPY ./dummy_data/server ./server

EXPOSE 8082
CMD gunicorn --bind 0.0.0.0:8082 server.wsgi:app