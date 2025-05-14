# GoHPC

> A lightweight High-Performance Computing toolkit for Go, providing parallel execution primitives, numerical routines, and more.

[![Release](https://img.shields.io/github/v/release/qcserestipy/gohpc?label=version&color=blue)](https://github.com/qcserestipy/gohpc/releases)
[![License](https://img.shields.io/github/license/qcserestipy/gohpc)](https://github.com/qcserestipy/gohpc/blob/main/LICENSE)

## Features

* **Generic Worker Pool Executor**: Dispatch arbitrary functions across multiple cores with ease.
* **Channel‑based Fan‑in/Fan‑out**: Simplify concurrent pipelines and task queues.
* **Per‑worker RNG Utilities**: Avoid global RNG contention in Monte Carlo workloads.
* **Extensible Architecture**: Add SIMD routines, parallel BLAS wrappers, distributed schedulers, and more.

## Installation

```bash
go get github.com/qcseresitpy/gophpc
```

## Quick Start

1. **Import the package**:

```go
import "github.com/qcseresitpy/gophpc/pkg/workerpool"
```

2. **Create a WorkerPoolExecutor**:

   ```go
   import (
       "runtime"
   )

   pool := workerpool.New[InputType, ResultType](runtime.NumCPU())
   ```

3. **Prepare inputs and work function**:

   ```go
   inputs := []InputType{...}
   work := func(in InputType) ResultType {
       // heavy computation here
       return result
   }
   ```

4. **Run in parallel and collect results**:

   ```go
   results := pool.Run(inputs, work)
   ```

## Example: Monte Carlo π Approximation
```bash
go build -o ./bin/pi cmd/example/main.go
./bin/pi
```

## Roadmap

* Distributed task scheduling over clusters

## Contributing

Contributions are welcome! Please open issues or pull requests in the [GitHub repository](https://github.com/qcserestipy/gophpc).

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
