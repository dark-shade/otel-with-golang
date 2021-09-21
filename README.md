# OpenTelemetry in Go with Elastic APM

This project showcase how to instrument a microservice written in [Go](https://golang.org/) using [OpenTelemetry](https://opentelemetry.io/), to produce telemetry data (traces and metrics) to [Elastic APM](https://www.elastic.co/apm).

## Talk
[Gopher Con Europe 2021](https://www.youtube.com/watch?v=NLXABIZ1gUQ)

## Run with the collector

Using this model, the Go application sends the traces and metrics to a collector that forwards them to Elastic APM.

```bash
docker compose -f run-with-collector.yaml up -d
```

## Run without the collector

Using this model, the Go application sends the traces and metrics directly to Elastic APM.

```bash
docker compose -f run-without-collector.yaml up -d
```

## Accessing Elastic APM

After executing the services you can reach the Elastic APM application in the following URL:

```bash
http://localhost:5601/app/apm
```

## Manually executing the Go application

Once everything is running there will periodic requests being sent to the microservice so you don't need to issue any requests by yourself. However, if you want to do it anyway just execute:

```bash
curl -X GET http://localhost:8888/hello
```

# License

This project is licensed under the [Apache 2.0 License](./LICENSE).
