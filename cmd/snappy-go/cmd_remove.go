package main

import (
	"fmt"

	"launchpad.net/snappy/snappy"
)

type cmdRemove struct {
}

func init() {
	var cmdRemoveData cmdRemove
	_, _ = parser.AddCommand("remove",
		"Remove a snapp part",
		"Remove a snapp part",
		&cmdRemoveData)
}

func (x *cmdRemove) Execute(args []string) (err error) {
	if !isRoot() {
		return ErrRequiresRoot
	}

	for _, part := range args {
		fmt.Printf("Removing %s\n", part)

		if err := snappy.Remove(part); err != nil {
			return err
		}
	}

	return nil
}