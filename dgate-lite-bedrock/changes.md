# Recent Changes

This document summarizes the recent changes made to the Gate proxy.

## Lite Mode Enhancements

### Bedrock Edition Support

- Added support for Minecraft: Bedrock Edition to the lite mode.
- The proxy can now listen for and forward UDP-based Bedrock connections.

### Unified Routing Configuration

- The `routes` in `config-lite.yml` now support a `protocol` field, which can be set to `tcp` (for Java Edition), `udp` (for Bedrock Edition), or `any` (for both).
- This allows for a single, unified routing configuration for both editions.

### Bedrock IP Forwarding

- A new `modifyBedrockIP` option has been added to the routes.
- When enabled for a Bedrock route, the proxy will forward the player's IP address to the backend server.

### Automatic Protocol-Specific Forwarding

- For routes with `protocol: any`, the proxy will automatically use the correct IP forwarding mechanism:
  - `proxyProtocol` for TCP (Java) connections.
  - `modifyBedrockIP` for UDP (Bedrock) connections.

### Configuration Updates

- The `defaultBackend` option for Bedrock has been removed in favor of the per-route `fallback` option.
- The `config-lite.yml` file has been updated with examples of the new routing configuration.

### Feature Parity

- The `fallback` and `cachePingTTL` options are now fully supported for Bedrock routes, providing the same level of resilience and performance as the Java routes.
