package partition

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Run the commandline specified by the args array chrooted to the given dir
var runInChroot = func(chrootDir string, args ...string) (err error) {
	fullArgs := []string{"/usr/sbin/chroot", chrootDir}
	fullArgs = append(fullArgs, args...)

	return runCommand(fullArgs...)
}

// FIXME: would it make sense to differenciate between launch errors and
//        exit code? (i.e. something like (returnCode, error) ?)
func runCommandImpl(args ...string) (err error) {
	if len(args) == 0 {
		return errors.New("no command specified")
	}

	// FIXME: use logger
	/*
		if debug == true {

			log.debug('running: {}'.format(args))
		}
	*/

	if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
		cmdline := strings.Join(args, " ")
		return fmt.Errorf("Failed to run command '%s': %s (%s)",
			cmdline, out, err)
	}
	return nil
}

// Run the command specified by args
// This is a var instead of a function to making mocking in the tests easier
var runCommand = runCommandImpl

// Run command specified by args and return the output
func runCommandWithStdoutImpl(args ...string) (output string, err error) {
	if len(args) == 0 {
		return "", errors.New("no command specified")
	}

	bytes, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", err
	}

	return string(bytes), err
}

// This is a var instead of a function to making mocking in the tests easier
var runCommandWithStdout = runCommandWithStdoutImpl