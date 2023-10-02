package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/codeclysm/extract"
)

// docker run image cmds
// ./chrun run redis [cmd]
func main() {
	switch os.Args[1] {
	case "child":
		child()
	case "run":
		parent()
	case "pull":
		image := os.Args[2]
		pullImage(image)
	default:
		panic("what?")
	}

}

func parent() {
	// create a child process
	// This will re-run this program as "chrun child <image> <cmd>" [run -> child]
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	// the child process will be in the defined namespaces
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
}

// chrun child [image] [cmd]
func child() {
	fmt.Println("You are now in a container...")
	// Now the child process will execute in an isolated environment
	image := os.Args[2]
	tar := fmt.Sprintf("./assets/%s.tar.gz", image)

	if _, err := os.Stat(tar); errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	cmd := ""
	if len(os.Args) > 3 {
		// command to run
		cmd = os.Args[3]
	} else {
		// default command (docker run redis) by default runs docker run redis redis-server
		buf, err := os.ReadFile(fmt.Sprintf("./assets/%s-cmd", image))
		if err != nil {
			panic(err)
		}
		cmd = string(buf)
	}

	dir := createTempDir(tar) // create a temperory directory to store the file system
	defer os.RemoveAll(dir)   // remove directory (clean up)
	must(unTar(tar, dir))     // extract the tar file to get image files
	chroot(dir, cmd)          // chroot into the temp directory

}

func pullImage(image string) {
	cmd := exec.Command("./pull.sh", image)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())
}

/**
NOTES-
- Pull Image .tar
- Create temporary directory
- Extract image (file system -alpine) from it
- Chroot into it
- Start the process
- On exit, delete directory

*/

func chroot(root string, call string) {
	//Hold onto old root
	oldrootHandle, err := os.Open("/")
	if err != nil {
		panic(err)
	}
	defer oldrootHandle.Close()

	//Create command
	fmt.Printf("Running %s in %s\n", call, root)
	cmd := exec.Command(call)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//New Root time
	must(syscall.Chdir(root))
	must(syscall.Chroot(root))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))

	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
	}

	//Go back to old root
	//So that we can clean up the temp dir
	must(syscall.Fchdir(int(oldrootHandle.Fd())))
	must(syscall.Chroot("."))

}

func createTempDir(name string) string {
	var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

	prefix := nonAlphanumericRegex.ReplaceAllString(name, "_")
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created %s\n", dir)
	return dir
}

func unTar(source string, dst string) error {
	// fmt.Printf("Extracting %s %s\n", source, dst)
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer r.Close()

	ctx := context.Background()
	return extract.Archive(ctx, r, dst, nil)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
