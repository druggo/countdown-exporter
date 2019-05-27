shorthash=`git rev-parse --short HEAD`

# Make sure to remove proto as a dependency if your repo doesn't use a protobuf file
all: docker-image

build-image:
	docker build --no-cache -t countdown-exporter-build-image:latest -f ./Dockerfile.build .
binary-for-docker-image: build-image
	docker run countdown-exporter-build-image:latest > ./countdown-exporter \
		&& chmod 0750 ./countdown-exporter
docker-image: binary-for-docker-image
	docker build -t countdown-exporter:${shorthash} .
	docker tag countdown-exporter:${shorthash} countdown-exporter:latest
