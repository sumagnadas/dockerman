# Container manager from scratch
This folder contains a basic container management system built from scratch, based on my own studies. As of now, following basic features are implemented:-
- Isolation of network, process and mountspace view using namespaces
- Unprivileged and privileged containers with isolated user namespaces
- Multi container management system based on this runtime
- gRPC daemon based on the CRI protocol definition

Future commits may include :-
- Configuration file based container builds
- Resource limiting using Cgroups

Current issues include, but are not limited to :-
- No synchronization for manipulation of the memory-based list of containers across multiple goroutines.
- No network connection with the host machine available or possible.

Topics learned or explored while building this project :-
- Go language
- Resource isolation and limiting in Linux-based OSes
- Nuances of privileged vs unprivileged containers on host systems
- Daemon based development

Started initially as a [small project](https://github.com/sumagnadas/small-projects/tree/master/container-from-scratch), but migrated to its own repo as the size of the project increased.
_NOTE: This is an educational project, so there may be many bugs that may or may not be fixed. If you find one, please inform me so that I may fix it and learn from the process._
## Usage

### dockman
```
Minimal container management system

Usage:
  dockman [command]

Available Commands:
  attach      Attaches the stdin, stdout and stderr of the command to the container
  completion  Generate the autocompletion script for the specified shell
  exec        Exec into a container with a command
  freeze      Freeze a container
  help        Help about any command
  info        Get info of a created container
  ps          Get a list of all containers
  remove      Remove a stopped container from daemon
  run         Create a container runtime with specified image and command
  start       Start a stopped container
  stop        Stop a container
  unfreeze    Unfreeze a container

Flags:
      --addr string   Address of the daemon (default "localhost:4033")
  -h, --help          help for dockman

Use "dockman [command] --help" for more information about a command.
```
### dockmand
```
Minimal container lifecycle and state management daemon

Usage:
  dockmand [flags]

Flags:
  -h, --help       help for dockmand
  -p, --port int   Specify an alternate port for the daemon (default 4033)
```
### dockmanc
```
Minimal container creation system

Usage:
  dockmanc [flags] <image> -- <command>

Flags:
  -h, --help          help for dockmanc
      --name string   Name of the container
      --user int      UID of the user
```

## Running the project
1. Install dependencies
```bash
go mod download
```
2. Get the minimal ubuntu-image FS for changing root (this requires `bash` in the container)
```bash
# $DOCKMAN_IMAGE_DIR is used to point to the dir containing the rootfs for the containers to base it on.
export DOCKMAN_IMAGE_DIR=$(pwd)
mkdir ubuntu && cd ubuntu && curl https://cloud-images.ubuntu.com/minimal/releases/noble/release/ubuntu-24.04-minimal-cloudimg-amd64-root.tar.xz -o ubuntu-fs.tar.xz && sudo tar -x -f ubuntu-fs.tar.xz
```
3. Build the binary (reqd. for rooted running for now) and run the container
```bash
# build the binary
go build -o dockmand daemon/main.go   # Daemon executable
go build -o dockmanc runtime/main.go  # Runtime creator executable
go build -o dockman # Main CLI app

# required by the daemon to find the runtime executable
export PATH=$PATH:$(pwd)

# Start the daemon
sudo -E PATH=$PATH ./dockmand 

# In a seperate terminal (privileged)
./dockman run ubuntu -it -- /bin/bash

# In a seperate terminal (unprivileged)
./dockman run ubuntu -uit -- /bin/bash
```
Use
```bash
./dockman ps
```
to find out what containers are running.

4. Entering into a container
```bash
./dockman run --name smth ubuntu -it -- /bin/bash # Works with both rooted and rootless container
./dockman exec  -- /bin/bash
```
5. a) Freezing and unfreezing a container
```bash
./dockman run --name smth ubuntu -it -- /bin/bash # requires a name or you can just use the id shown in "dockman ps"
./dockman freeze smth
./dockman ps # Check status is frozen
./dockman unfreeze smth
```

5. b) Stopping and starting a container
```bash
./dockman run --name smth ubuntu -- /bin/bash # requires a name or you can just use the id shown in "dockman ps"
./dockman stop smth
./dockman ps # Check status is stopped
./dockman start smth
```

## Some major sources I used for studying
- [Liz Rice's Container from Scratch](https://www.youtube.com/watch?v=8fi7uSYlOdc)
- [Red Hat Blog's posts on container](https://www.redhat.com/en/blog/mount-namespaces)
- [Jerome Petazzoni's talk on containers](https://www.youtube.com/watch?v=sK5i-N34im8)
- Docker, CRI API
- Random strangers on Reddit, Stack Overflow and Medium whose explanation solidified the foundations more from the above sources.
