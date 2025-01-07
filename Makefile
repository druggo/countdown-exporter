shorthash=`git rev-parse --short HEAD`

# Make sure to remove proto as a dependency if your repo doesn't use a protobuf file
all: docker-image

docker-image:
	docker build -t countdown-exporter:${shorthash} .
	docker tag countdown-exporter:${shorthash} countdown-exporter:latest
