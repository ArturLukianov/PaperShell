# PaperShell agent

Simple Powershell agent for Adaptix C2

Installation:

```
cd listener_papershell_http
make
```
Copy dist to AdaptixC2/dist/extenders/listener_papershell_http

```
cd papershell_agent
make
```
Copy dist to AdaptixC2/dist/extenders/papershell_agent

Add new extenders to AdaptixC2 profile.json:

```json
"extenders": [
      "extenders/beacon_listener_http/config.json",
      "extenders/beacon_listener_smb/config.json",
      "extenders/beacon_listener_tcp/config.json",
      "extenders/beacon_agent/config.json",
      "extenders/gopher_listener_tcp/config.json",
      "extenders/gopher_agent/config.json",

      "extenders/listener_papershell_http/config.json",
      "extenders/papershell_agent/config.json"
]
```