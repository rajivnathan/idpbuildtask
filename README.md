# IDP Build Task POC

The IDP Build Task POC runs a Kube Job that builds the Codewind projects in a build container and update the Kube Persistent Volume Mount of the project with the build artifacts. 

This allows the project run time container to be free of build task tools and utilities.

Run `GOOS=linux go build -o idcbuildtask` to build the go command line binary for the linux architecture

## Usage:
Run `idcbuildtask <project task name> <project name>` to kick off a Kube Job for a Liberty Build Task.