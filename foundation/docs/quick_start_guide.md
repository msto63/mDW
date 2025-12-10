# mDW Foundation - Quick Start Guide

**Version**: v1.0.0  
**Status**: Production Ready  
**Letzte Aktualisierung**: 2025-07-26  

## 🚀 **Schnellstart in 5 Minuten**

### **1. Repository Setup**
```bash
# Clone the repository
git clone <repository-url>
cd foundation

# Initialize Go module (falls noch nicht vorhanden)
go mod init github.com/msto63/mDW/foundation
go mod tidy

# Verify build
go build ./pkg/...
```

### **2. Erste TCOL Commands**
```tcol
# Basis-Syntax
CUSTOMER.CREATE name="Example Corp" type="B2B"
CUSTOMER:12345                                    // Show customer
CUSTOMER[city="Berlin"].LIST                      // Filter
CUSTOMER:12345:email="new@example.com"            // Update

# Erweiterte Features
CUST.CR name="Short Corp"                        // Abbreviation
INVOICE[unpaid=true,age>30].SEND-REMINDER         // Complex filter
```

### **3. Foundation Module Usage**
```go
package main

import (
    mdwerrors "github.com/msto63/mDW/foundation/pkg/core/errors"
    mdwlog "github.com/msto63/mDW/foundation/pkg/core/log"
    mdwmathx "github.com/msto63/mDW/foundation/pkg/utils/mathx"
    mdwstringx "github.com/msto63/mDW/foundation/pkg/utils/stringx"
    mdwvalidationx "github.com/msto63/mDW/foundation/pkg/utils/validationx"
)

func main() {
    // Logging
    logger := mdwlog.New(mdwlog.WithLevel(mdwlog.LevelInfo))
    logger.Info("Application started")
    
    // String operations
    cleaned := mdwstringx.ToTitleCase("hello world")
    logger.Info("Processed string", mdwlog.String("result", cleaned))
    
    // Math operations
    price, _ := mdwmathx.NewDecimal("29.99")
    tax := price.Multiply(mdwmathx.MustNewDecimal("0.19"))
    total := price.Add(tax)
    logger.Info("Price calculation", mdwlog.String("total", total.String()))
    
    // Validation
    result := mdwvalidationx.ValidateStruct(struct{
        Email string `validate:"required,email"`
    }{Email: "user@example.com"})
    
    if !result.Valid {
        logger.Error("Validation failed", mdwlog.Any("errors", result.Errors))
        return
    }
    
    logger.Info("Application completed successfully")
}
```

## 📚 **Core Concepts**

### **Import Aliasing Pattern**
Alle mDW Foundation Module verwenden das `mdw{module}` Aliasing:
```go
import (
    mdwerror "pkg/core/error"        // Error framework
    mdwerrors "pkg/core/errors"      // Error API
    mdwlog "pkg/core/log"           // Logging
    mdwconfig "pkg/core/config"     // Configuration
    mdwi18n "pkg/core/i18n"         // Internationalization
    mdwvalidation "pkg/core/validation" // Validation framework
    
    mdwstringx "pkg/utils/stringx"   // String utilities
    mdwmathx "pkg/utils/mathx"       // Math utilities
    mdwmapx "pkg/utils/mapx"         // Map utilities
    mdwslicex "pkg/utils/slicex"     // Slice utilities
    mdwtimex "pkg/utils/timex"       // Time utilities
    mdwfilex "pkg/utils/filex"       // File utilities
    mdwvalidationx "pkg/utils/validationx" // Validation implementations
)
```

### **Error Handling Pattern**
```go
// Standard error creation
if input == "" {
    return mdwerrors.ValidationFailed("mymodule", "input", input, "cannot be empty")
}

// Math errors
if divisor.IsZero() {
    return mdwerrors.MathxDivisionByZero("divide")
}

// Custom errors
err := mdwerrors.NewErrorBuilder("mymodule").
    Operation("my_operation").
    Message("Something went wrong").
    Detail("context", additionalInfo).
    Severity(mdwerror.SeverityHigh).
    Build()
```

### **Validation Pattern**
```go
// Use framework + implementation
import (
    "pkg/core/validation"      // Framework
    "pkg/utils/validationx"    // Concrete validators
)

// Validate single field
result := mdwvalidationx.Email().Validate("user@example.com")
if !result.Valid {
    return result.ToError()
}

// Validate struct
result := mdwvalidationx.ValidateStruct(myStruct)
if !result.Valid {
    // Handle validation errors
    for _, err := range result.Errors {
        log.Printf("Field %s: %s", err.Field, err.Message)
    }
}
```

## 🏗️ **Architektur-Übersicht**

### **Module-Kategorien**

#### **Core Modules** (Infrastructure)
- **error/errors**: Two-layer error system
- **log**: Structured logging mit 7 levels
- **config**: TOML/YAML configuration
- **i18n**: Internationalization
- **validation**: Validation framework (nur interfaces)

#### **Utility Modules** (Implementations)
- **stringx**: Unicode-safe string operations
- **mathx**: Precise decimal arithmetic
- **mapx**: Generic map operations
- **slicex**: Functional slice operations
- **timex**: Time/date utilities
- **filex**: File operations
- **validationx**: Validation implementations

#### **TCOL Modules** (Command Language)
- **tcol**: Main engine
- **lexer**: Tokenization
- **parser**: Parsing
- **ast**: Abstract syntax tree
- **registry**: Object/method registry
- **executor**: Command execution
- **client**: Service client

### **Dependency Flow**
```
Application
    ↓
TCOL Engine ←→ Core Modules ←→ Utility Modules
    ↓              ↓              ↓
External      Foundation     Concrete
Services      Framework    Implementations
```

## 📖 **Wichtige Guides**

### **Für Benutzer:**
- 📘 **TCOL_USER_GUIDE.md** - Complete TCOL syntax and usage
- 📗 **ERROR_HANDLING_GUIDE.md** - Error system reference
- 📙 **VALIDATION_ARCHITECTURE.md** - Validation patterns

### **Für Entwickler:**
- 🔧 **TCOL_DEVELOPER_GUIDE.md** - Technical implementation
- 📋 **programming_guidelines.md** - Coding standards
- 🎯 **examples/integration_examples.go** - Real-world patterns

### **Für Architekten:**
- 🏗️ **PROJECT_STATUS_FINAL.md** - Complete project overview
- 📊 **DEVELOPMENT_HISTORY.md** - Development journey
- 🔄 **VALIDATION_ARCHITECTURE.md** - System boundaries

## 🔍 **Häufige Use Cases**

### **1. E-commerce Order Processing**
```go
// Siehe examples/integration_examples.go - Example 1
processor, _ := NewOrderProcessor("0.08", "5.99") // tax rate, shipping
result, err := processor.ProcessOrder(order)
if err != nil {
    log.Printf("Order failed: %v", err)
    return
}
fmt.Printf("Total: $%s", result.Total)
```

### **2. Data Validation Pipeline**
```go
// Siehe examples/integration_examples.go - Example 2  
pipeline := NewDataPipeline()
processed, err := pipeline.ProcessRecord(record)
if err != nil {
    log.Printf("Validation failed: %v", err)
    return
}
fmt.Printf("Score: %d/100", processed.ValidationScore)
```

### **3. TCOL Business Workflows**
```go
// Siehe examples/tcol_business_examples.go
demo := NewBusinessExamples()
demo.CustomerLifecycleManagement()
demo.InvoiceProcessingWorkflow()
demo.ProjectManagementWorkflow()
```

### **4. Foundation Module Integration**
```go
// Siehe examples/tcol_integration_demo.go
demo := NewIntegrationDemo()
demo.ErrorHandlingIntegration()
demo.LoggingIntegration()
demo.UtilityIntegration()
```

## 🧪 **Testing**

### **Run All Tests**
```bash
# Basic test run
go test ./...

# Verbose with coverage
go test -v -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. -benchmem ./...
```

### **Module-Specific Testing**
```bash
# Test specific module
go test -v ./pkg/utils/stringx/...

# Test with race detection
go test -race ./pkg/core/log/...

# Benchmark specific module
go test -bench=. ./pkg/utils/mathx/...
```

## ⚡ **Performance Tips**

### **String Operations**
```go
// Use stringx for better performance
result := mdwstringx.PadLeft(input, 10, ' ')  // Optimized
// vs strings.Repeat(" ", 10-len(input)) + input  // Slower
```

### **Math Operations**
```go
// Reuse decimal objects
price := mdwmathx.MustNewDecimal("29.99")
tax := price.Multiply(taxRate)  // Creates new instance
price.Free()  // Return to pool when done
```

### **Validation**
```go
// Chain validators efficiently
chain := mdwvalidationx.NewValidatorChain().
    AddFunc(mdwvalidationx.Required()).
    AddFunc(mdwvalidationx.Email())
result := chain.Validate(input)
```

## 🔒 **Security Considerations**

### **Input Validation**
```go
// Always validate input
if err := mdwvalidationx.ValidateStruct(userInput); err != nil {
    return mdwerrors.ValidationFailed("api", "input", userInput, "invalid input")
}
```

### **Error Information**
```go
// Don't expose sensitive data in errors
return mdwerrors.InvalidInput("auth", "login", "[REDACTED]", "valid credentials")
```

### **Logging**
```go
// Use structured logging, avoid sensitive data
logger.Info("User action", 
    mdwlog.String("user_id", userID),        // OK
    mdwlog.String("action", "login"),        // OK
    // mdwlog.String("password", password),  // NEVER!
)
```

## 📞 **Support & Resources**

### **Documentation Locations**
- `/docs/` - Additional guides
- `/examples/` - Practical examples
- `/pkg/*/doc.go` - Module-specific documentation
- `/*.md` - Project guides

### **Example Code**
- `examples/integration_examples.go` - Multi-module integration
- `examples/tcol_business_examples.go` - Business workflows  
- `examples/tcol_integration_demo.go` - Advanced integration
- `pkg/tcol/examples/basic_syntax.go` - TCOL syntax

### **Troubleshooting**
1. **Build Issues**: Verify `go mod tidy` ausgeführt
2. **Import Errors**: Check mdw{module} aliasing pattern
3. **Test Failures**: Run `go test -v` für details
4. **Performance**: Use `-benchmem` für memory profiling

---

**Los geht's!** 🚀  
Mit diesem Guide kannst du sofort mit der mDW Foundation entwickeln. Für detailliertere Informationen siehe die spezifischen Guides in der Dokumentation.