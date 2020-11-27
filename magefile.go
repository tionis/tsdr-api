//+build mage

package main

import (
    "fmt"

    "github.com/magefile/mage/mg"
    "github.com/magefile/mage/sh"
)

type Backend mg.Namespace

func (Backend) Build() error {
    fmt.Println("compiling binary api")
    return sh.Run("go", "build", "-o", "dist/api", ".")
}

func Build() {
    mg.SerialDeps(Backend.Build)
}
