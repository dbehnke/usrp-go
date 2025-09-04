package main

import (
    "context"
    "fmt"
    "os"

    dagger "dagger.io/dagger"
)

func main() {
    ctx := context.Background()

    // Connect to Dagger engine using default settings (DAGGER_HOST env or local)
    c, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
    if err != nil {
        fmt.Fprintf(os.Stderr, "dagger connect: %v\n", err)
        os.Exit(2)
    }
    defer c.Close()

    // Mount repository into /src in the container
    src := c.Host().Directory(".")

    // Use Ubuntu so /usr/bin/env bash exists for the validator script
    ctr := c.Container().From("ubuntu:24.04").WithDirectory("/work", src).WithWorkdir("/work")

    // Run the validator script (ensure executable). Use /bin/sh -c to chain commands.
    cmd := []string{"/bin/sh", "-c", "chmod +x test/containers/test-validator/run-integration-tests.sh && test/containers/test-validator/run-integration-tests.sh"}
    res := ctr.WithExec(cmd)

    // Always try to print stdout/stderr for debugging
    if out, err := res.Stdout(ctx); err == nil && out != "" {
        fmt.Print(out)
    }
    if errOut, err := res.Stderr(ctx); err == nil && errOut != "" {
        fmt.Fprint(os.Stderr, errOut)
    }

    exitCode, err := res.ExitCode(ctx)
    if err != nil {
        // show the error and exit
        fmt.Fprintf(os.Stderr, "failed to get exit code: %v\n", err)
        os.Exit(3)
    }

    if exitCode != 0 {
        fmt.Fprintf(os.Stderr, "validator failed with exit code %d\n", exitCode)
        os.Exit(exitCode)
    }

    fmt.Println("Integration validator succeeded")
}
