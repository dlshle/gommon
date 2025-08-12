# gommon

A collection of common Go utilities and libraries.

## Overview

Gommon is a Go module that provides a set of utilities and libraries commonly used in Go applications. It includes implementations for:

- **Async**: Asynchronous programming utilities including futures, barriers, and task queues
- **Data Structures**: Custom data structures like hash sets, linked lists, queues, and insertion lists
- **HTTP**: Enhanced HTTP client with interceptors and builders
- **Connection**: Connection pooling utilities
- **IO**: Input/Output utilities
- **IOC**: Inversion of control container
- **Logging**: Logging utilities
- **Error Handling**: Extended error handling capabilities

## Installation

```bash
go get github.com/dlshle/gommon
```

## Usage

Import the specific packages you need:

```go
import (
    "github.com/dlshle/gommon/async"
    "github.com/dlshle/gommon/data_structures"
    "github.com/dlshle/gommon/http"
    // ... other imports
)
```

## Examples

Check the test files in each directory for usage examples of the various utilities.

## License

This project is licensed under the MIT License - see the LICENSE file for details.