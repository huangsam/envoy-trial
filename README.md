# Envoy Trial

Testing out Envoy in a variety of scenarios.

The first one is UDP connections over the UDP proxy.

The Envoy proxy runs on external ports 9901 (for admin) and 10161 (for UDP). Whenever
it receives UDP traffic, it forwards UDP traffic over to an internal port 161 for the
associated Go listener. The listener is designed to process requests concurrently.

## Getting started

```shell
# In one terminal
docker-compose up --build

# In another terminal
echo "Test message" | nc -u -w 1 127.0.0.1 10161
```

## Resources

Documentation

- [Docker usage](https://www.envoyproxy.io/docs/envoy/latest/start/docker)
- [UDP overview](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/udp_filters/udp_proxy)
- [Admin interface](https://www.envoyproxy.io/docs/envoy/latest/start/quick-start/admin)
- [envoy/issues/21617](https://github.com/envoyproxy/envoy/issues/21617)

Schemas

- [UDP proto](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/udp/udp_proxy/v3/udp_proxy.proto)
- [Cluster proto](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/cluster/v3/cluster.proto)
