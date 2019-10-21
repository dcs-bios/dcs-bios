# DCS-BIOS

## Repository Structure

* `src/hub-backend`: DCS-BIOS Hub backend (written in Go)
* `src/hub-frontend`: Web frontend (based on React, written in TypeScript)

## Development

To run the backend application:
````
cd src
.\build_and_run.cmd
````

To run the frontend application:
````
cd src/hub-frontend
npm start
````

Ignore the built-in webserver on port 5010, it can only serve the frontend when it has been built using `npm build` and moved to the right place. For development, use `http://localhost:3000` instead, which runs the webpack development server. It automatically reloads when you change a source file and provides nice error messages.

The frontend will detect that it is being served from port 3000, assume it is in development mode, and still direct API requests to port 5010.


## Building a Release

Prerequisites:

| Tool | Where to download | Purpose |
| --- | --- | --- | 
| `go` | https://golang.org/ | The DCS-BIOS Hub is implemented in the Go programming language. |
| `npm` | https://nodejs.org/ | To build the frontend web application, you will need the `npm` package manager, which is part of node.js. |
| Wix Toolkit | https://wixtoolset.org/releases/ | Builds the .msi installer |
| `go-msi` | run `go install github.com/mh-cbon/go-msi` | Configures and invokes the Wix Toolkit |

To create an MSI installer, run `npm install` in the `src\hub-frontend` directory, then run `release-build.cmd` from the repository root.
If your code checkout is not on drive C:, you have to set the `TMP` environment variable to point to a directory on the same drive as your code.
