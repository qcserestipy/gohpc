> # GoHPC

> A lightweight High-Performance Computing toolkit for Go, providing parallel execution primitives, numerical routines, and more.

[![Release](https://img.shields.io/github/v/release/qcserestipy/gohpc?label=version\&color=blue)](https://github.com/qcserestipy/gohpc/releases)
[![License](https://img.shields.io/github/license/qcserestipy/gohpc)](https://github.com/qcserestipy/gohpc/blob/main/LICENSE)

## Features

* **Generic Worker Pool Executor**: Dispatch arbitrary functions across multiple cores with ease, now fully context-aware and cancelable.
* **Channel‑based Fan‑in/Fan‑out**: Simplify concurrent pipelines and task queues.
* **Per‑worker RNG Utilities**: Avoid global RNG contention in Monte Carlo workloads.
* **Extensible Architecture**: Add SIMD routines, parallel BLAS wrappers, distributed schedulers, and more.
* **Graceful Shutdown**: All execution primitives respect `context.Context` cancellation and timeouts.

## Installation

```bash
go get github.com/qcserestipy/gohpc
```

## Quick Start

1. **Import the package**:

```go
import (
    "context"
    "github.com/qcserestipy/gohpc/pkg/workerpool"
)
```

2. **Create a WorkerPoolExecutor** (optionally customize worker count):

```go
// default uses runtime.NumCPU() workers
pool := workerpool.New[InputType, ResultType](
    workerpool.WithWorkers(4), // override to 4 workers
)
```

3. **Prepare inputs, context, and work function**:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

inputs := []InputType{ /* ... */ }
work := func(ctx context.Context, in InputType) ResultType {
    // heavy computation, periodically check ctx.Done() if desired
    return result
}
```

4. **Run in parallel and collect results**:

```go
results, err := pool.Run(ctx, inputs, work)
if err != nil {
    // handle cancellation or error
}
```

## Example: Monte Carlo π Approximation

```bash
go build -o ./bin/pi cmd/example/main.go
./bin/pi -n 10000000000
```

In code:

```go
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer cancel()

pool := workerpool.New[Task, float64](workerpool.WithWorkers(runtime.NumCPU()))
results, err := pool.Run(ctx, tasks, work)
if err != nil {
    logrus.Warnf("Computation cancelled: %v", err)
    return
}
```

## Roadmap

* Distributed task scheduling over clusters
* SIMD intrinsic support and low-level tuning primitives

## Contributing

Contributions are welcome! Please open issues or pull requests in the [GitHub repository](https://github.com/qcserestipy/gohpc).

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
