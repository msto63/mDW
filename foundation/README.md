# mDW Foundation

The mDW Foundation is a comprehensive Go library providing core functionality for the Trusted Business Platform. It includes essential modules for configuration, error handling, internationalization, logging, validation, utilities, and the Terminal Command Object Language (TCOL).

## Modules

### Core Modules
- **config**: Configuration management with file watching and validation
- **error**: Structured error handling with codes and severity levels  
- **errors**: Shared error utilities for consistent error patterns
- **i18n**: Internationalization support with TOML/YAML translations
- **log**: Structured logging with multiple output formats
- **validation**: Validation framework with chaining and custom rules

### Utility Modules
- **filex**: File system operations and utilities
- **mapx**: Map manipulation and utility functions
- **mathx**: Precise decimal arithmetic for financial calculations
- **slicex**: Slice operations and transformations
- **stringx**: String manipulation and validation
- **timex**: Comprehensive time utilities and business day calculations
- **validationx**: Extended validation rules and patterns

### TCOL Module
- **tcol**: Terminal Command Object Language parser and interpreter

## Installation

```bash
go get github.com/msto63/mDW/foundation
```

## Usage

```go
import (
    "github.com/msto63/mDW/foundation/core/log"
    "github.com/msto63/mDW/foundation/utils/stringx"
    "github.com/msto63/mDW/foundation/tcol"
)
```

## Testing

```bash
go test ./...
```

## Documentation

See the `docs/` directory for comprehensive documentation.

## License

See LICENSE file for details.