# Container runtime from scratch
This folder contains a basic container runtime management system built from scratch, based on my own studies. As of now, following basic features are implemented:-
- Isolation of network, process and mountspace view using namespaces
- Rooted (No user isolation)
- Rootless (user isolation)
- Multi container management system based on this runtime

Future commits will include :-
- Configuration file based container builds
- Resource limiting using Cgroups

Topics learned or explored while building this project :-
- Go language
- Resource isolation and limiting in Linux-based OSes
- Nuances of privileged vs unprivileged containers on host systems
- Daemon based development

Started initially as a [small project](https://github.com/sumagnadas/small-projects/tree/master/container-from-scratch), but migrated to its own repo as the size of the project increased.
_NOTE: This is an educational project, so there may be many bugs that may or may not be fixed. If you find one, please inform me so that I may fix it and learn from the process._
## Usage
```
Minimal container management system

Usage:
  dockerman [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  daemon      Launch a daemon to manage containers.
  exec        Execute a command inside a container.
  freeze      Freeze a running container
  help        Help about any command
  ps          Get a list of running containers
  run         Run a container runtime with image and command (attaches the stdin, stdout and stderr of the command to shell)
  unfreeze    Unfreeze a freezed container

Flags:
  -h, --help   help for dock

Use "dockerman [command] --help" for more information about a command.
```
## Running the project
1. Install dependencies
```bash
go mod download
```
2. Get the minimal ubuntu-image FS for changing root (this requires `bash` in the container)
```bash
mkdir ubuntu && cd ubuntu && curl https://cloud-images.ubuntu.com/minimal/releases/noble/release/ubuntu-24.04-minimal-cloudimg-amd64-root.tar.xz -o ubuntu-fs.tar.xz && sudo tar -x -f ubuntu-fs.tar.xz
```
3. Build the binary (reqd. for rooted running for now) and run the container
```bash
go build -o dockerman
# Command format
# dockerman run <image> <cmd> <args...>
./dockerman daemon & # start the daemon
sudo ./dockerman run ubuntu -- /bin/bash # for an interactive shell
sudo ./dockerman run ubuntu -- /bin/bash -c date # for running a command in a container
```
Use
```bash
./dockerman ps
```
to find out what containers are running.

3. (Rootless) This can be run as an unprivileged user.
```bash
sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0 # Ubuntu-specific
# This disables apparmor protection for restricting unprivileged user namespaces, used in many exploits.
# Do at your own risk
./dockerman run ubuntu -- /bin/bash
```
4. Entering into a container (Requires root)
```bash
./dockerman run --name smth ubuntu -- /bin/bash # Works with both rooted and rootless container
sudo ./dockerman exec smth -- /bin/bash
```
5. Freezing and unfreezing a container
```bash
./dockerman run --name smth ubuntu -- /bin/bash # requires a name
./dockerman freeze smth
./dockerman ps # Check running status is frozen
./dockerman unfreeze smth
```

## Some major sources I used for studying
- [Liz Rice's Container from Scratch](https://www.youtube.com/watch?v=8fi7uSYlOdc)
- [Red Hat Blog's posts on container](https://www.redhat.com/en/blog/mount-namespaces)
- [Jerome Petazzoni's talk on containers](https://www.youtube.com/watch?v=sK5i-N34im8)
- Random strangers on Reddit and Medium whose explanation solidified the foundations more from the above sources.
