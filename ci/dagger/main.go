package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

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

    // Mount the repository root into the container. When running from ci/dagger the
    // repository root is two levels up (../..). Compute an absolute path to be safe.
    cwd, err := os.Getwd()
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to get cwd: %v\n", err)
        os.Exit(4)
    }
    repoRoot := cwd + "/../.."
    // Prefer an absolute path
    repoRootAbs, err := filepath.Abs(repoRoot)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to resolve repo root: %v\n", err)
        os.Exit(5)
    }
    src := c.Host().Directory(repoRootAbs)

    // Use Ubuntu so /usr/bin/env bash exists for the validator script
    ctr := c.Container().From("ubuntu:24.04").WithDirectory("/work", src).WithWorkdir("/work")

    // Diagnostic: list and print the validator script first to inspect its contents
    probe := ctr.WithExec([]string{"/bin/sh", "-c", "echo '--- listing ---' && ls -la test/containers/test-validator && echo '--- cat ---' && sed -n '1,200p' test/containers/test-validator/run-integration-tests.sh || true"})

    if out, err := probe.Stdout(ctx); err == nil && out != "" {
        fmt.Print(out)
    }
    if errOut, err := probe.Stderr(ctx); err == nil && errOut != "" {
        fmt.Fprint(os.Stderr, errOut)
    }

    // Run the validator script via sh to avoid depending on executable bit inside the
    // Dagger container image.
    cmd := []string{"/bin/sh", "-c", "sh test/containers/test-validator/run-integration-tests.sh"}
    res := ctr.WithExec(cmd)

    // Debug: try to fetch stdout/stderr and any exec error
    out, outErr := res.Stdout(ctx)
    if outErr == nil && out != "" {
        fmt.Print(out)
    }
    errOut, errErr := res.Stderr(ctx)
    if errErr == nil && errOut != "" {
        fmt.Fprint(os.Stderr, errOut)
    }

    exitCode, err := res.ExitCode(ctx)
    if err != nil {
        // show the error and include stdout/stderr fetch errors
        fmt.Fprintf(os.Stderr, "failed to get exit code: %v\n", err)
        if outErr != nil { fmt.Fprintf(os.Stderr, "stdout fetch error: %v\n", outErr) }
        if errErr != nil { fmt.Fprintf(os.Stderr, "stderr fetch error: %v\n", errErr) }
        os.Exit(3)
    }

    if exitCode != 0 {
        fmt.Fprintf(os.Stderr, "validator failed with exit code %d\n", exitCode)
        os.Exit(exitCode)
    }

    fmt.Println("Integration validator succeeded")
}
