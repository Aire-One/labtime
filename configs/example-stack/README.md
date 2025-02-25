# Demo Prometheus/Grafana stack with Labtime metrics

This is a Docker compose stack example to run Prometheus with Grafana to play with the Labtime metrics.

Labtime should be running  on the host (e.g., `make dev` from the project root).

Prometheus Scrapper is autoconfigured from the `prometheus-data/prometheus.yaml` configuration file and listens to the host Labtime exporter with the docker compose `external_hosts`.

Grafana is autoconfigured with configurations from the `grafana-provisioning` folder. The default Datasource is Prometheus, and a dashboard with Labtime metrics is created.

Connect to Grafana from <http://localhost:3000>, user: `admin` password: `admin`
