## Audio Router

The audio-router is a small service in this repository responsible for routing and converting audio streams used by the integration tests and local development environment.

Summary
- Purpose: accept audio input from various sources (UDP, files, or mocked services), convert or transcode as needed, and forward to downstream services (e.g. Allstar mock, USRP adapters).
- Location: implementation and tests live under `test/containers` and `pkg/usrp`.

Running locally
- The integration stack in `test/integration/docker-compose.yml` includes the audio-router service.
- To run the integration stack locally (requires Docker with Compose plugin or the repo shim), use the repo tasks:

  - `just test-integration` — runs the integration tasks and validator.

Configuration
- Build context: the `audio-router` service uses a build context located at the repository root so source files are available to the Docker build; see `test/integration/docker-compose.yml` for details.
- Ports: the router listens on configured UDP ports (see `test/containers/*` mocks and `test/tilt` k8s manifests for examples).

Development notes
- Tests: the `test/containers/test-validator` script runs end-to-end checks that the audio-router and other services interoperate.
- Logging: run the router locally or inside the integration stack and use `docker logs` to inspect runtime behavior.

Troubleshooting
- If Compose build fails with "lstat ... no such file or directory", verify build contexts in `test/integration/docker-compose.yml` point to the repository root.
- If a container fails to start due to a mount type mismatch for `prometheus.yml` or similar, ensure the file exists in `test/integration/configs/` and is a regular file, not a directory.

References
- test/integration/docker-compose.yml
- test/containers/test-validator/run-integration-tests.sh
- test/containers/audio-router (implementation and Dockerfile)

If you'd like, I can expand this page with architecture diagrams, example UDP packets, or a quickstart showing how to run the router and stream audio into it locally.
