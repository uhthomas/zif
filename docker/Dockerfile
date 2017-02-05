FROM gliderlabs/alpine:3.4

ARG bind=0.0.0.0:5050
ARG http=127.0.0.1:8080

RUN apk update && \
	apk add wget make go git gcc musl-dev && \
	mkdir -p /var/lib/zif && \
	cd /var/lib/zif && \
	git clone https://github.com/zif/zif src && \
	cd src && make && cd ../ && \
	cp src/bin/zifd . && \
	wget --no-check-certificate https://willhuxtable.com/zifd.toml && \
	chmod +x zifd

EXPOSE 8080
EXPOSE 5050
CMD cd /var/lib/zif && ./zifd --bind ${bind} --http ${http}
