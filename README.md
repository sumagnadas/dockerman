# Container runtime from scratch
This folder contains a basic container runtime built from scratch, based on my own studies and Liz Rice's video on [Containers From Scratch](https://www.youtube.com/watch?v=8fi7uSYlOdc). As of now, following basic features are implemented:-
- Isolation of network, process and mountspace view using namespaces
- Rooted (No user isolation)

Future commits will include :-
- Rootless (user isolation)
- Multi container management system based on this runtime
- Configuration file based container builds
- Resource limiting using Cgroups

Topics learned or explored while building this project :-
- Go language
- Resource isolation and limiting in Linux-based OSes
- Nuances of privileged vs unprivileged containers on host systems

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
go build -o dock main.go
# Command format
# dock run <image> <cmd> <args...>
sudo ./dock run ubuntu -- /bin/bash # for an interactive shell
sudo ./dock run ubuntu -- /bin/bash -c date # for running a command in a container
```
3. (Rootless) This can be run as an unprivileged user.
```bash
sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0 # Ubuntu-specific
# This disables apparmor protection for restricting unprivileged user namespaces, used in many exploits.
# Do at your own risk
./dock run ubuntu -- /bin/bash
```
