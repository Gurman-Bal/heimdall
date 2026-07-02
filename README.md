# Heimdall (In progress)

Heimdall is a lightweight log monitoring tool for TrueNAS systems. It allows users to configure log sources, define regex-based rules, and monitor events through a real-time web interface.

The backend is written in Go and handles source management, rule evaluation, and event streaming using Server-Sent Events (SSE). The frontend uses vanilla HTML, CSS, and JavaScript with a modular structure to keep the application lightweight and easy to extend.

## Features

- Real-time event streaming
- Configurable log sources
- Regex-based detection rules
- Severity levels (info, warning, critical)
- Web UI for managing sources and rules
- Modular frontend and backend architecture

## Future Improvements

- Notifications and alerts
- Automated remediation actions
- Authentication and user management
- Historical event storage
- Additional source integrations