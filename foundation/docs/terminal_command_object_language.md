# TCOL – Terminal Command Object Language

## Business and Technical Concept

### 1  Executive Summary

TCOL (Terminal Command Object Language) is an object-oriented command language for a TUI-based microservice application system. It combines the efficiency of classic terminal commands with modern object-oriented paradigms and enables intuitive, shorten-as-you-like commands for enterprise applications.

**Key features:**

* Fully object-based syntax
* Intelligent abbreviations down to uniqueness
* Consistent method notation
* Comprehensive alias system
* Multi-service support

### 2  Architecture Overview

```
┌─────────────────────┐
│   TUI Client        │
│  ┌───────────────┐  │
│  │ Input Parser  │  │
│  └───────┬───────┘  │
│          │          │
│  ┌───────▼───────┐  │
│  │ Alias Resolver│  │
│  └───────┬───────┘  │
│          │          │
│  ┌───────▼───────┐  │
│  │Command Builder│  │
│  └───────┬───────┘  │
└──────────┼──────────┘
           │ gRPC
┌──────────▼──────────┐
│ Application Server  │
│  ┌───────────────┐  │
│  │Command Parser │  │
│  └───────┬───────┘  │
│          │          │
│  ┌───────▼───────┐  │
│  │  Validator    │  │
│  └───────┬───────┘  │
│          │          │
│  ┌───────▼───────┐  │
│  │ Router/Exec   │  │
│  └───────┬───────┘  │
└──────────┼──────────┘
           │ gRPC
┌──────────▼──────────┐
│   Microservices     │
└─────────────────────┘
```

### 3  Syntax Specification

#### 3.1 Grammar (EBNF)

```ebnf
command        = object_command | system_command | alias_command
object_command = object_spec "." method_name [parameters]
system_command = system_object "." method_name [parameters]
alias_command  = alias_name [parameters]

object_spec    = object_type "[" selector "]" | object_type ":" identifier
object_type    = IDENTIFIER
selector       = "*" | identifier | filter_expression
identifier     = ALPHANUMERIC+
filter_expression = condition {"," condition}
condition      = field_name operator value

method_name    = IDENTIFIER {"-" IDENTIFIER}
parameters     = parameter {"," parameter}
parameter      = param_name "=" param_value
param_value    = string | number | boolean | array

system_object  = "SYSTEM" | "QUERY" | "REPORT" | "BATCH" |
                 "TRANSACTION" | "VARIABLE" | "MACRO" |
                 "ALIAS" | "HELP"
```

#### 3.2 Abbreviation Rules

**Algorithm for abbreviation detection**

1. **Token-based abbreviation** – Each dash-separated token can be abbreviated independently

   ```
   SHOW-SERVICE-STATUS → SH-SERV-ST → S-S-S
   ```

2. **Uniqueness check**

   * The parser maintains a prefix tree (trie) containing all known commands.
   * An abbreviation must resolve to exactly one command.
   * If ambiguous: error plus suggestion list.

3. **Context-aware resolution**

   * Object methods are resolved in the context of their object type.
   * System methods are resolved globally.

### 4  Object Model

#### 4.1 Base Object Types

```
OBJECT
├── BUSINESS_OBJECT
│   ├── CUSTOMER
│   ├── INVOICE
│   ├── ORDER
│   └── PRODUCT
├── SYSTEM_OBJECT
│   ├── SYSTEM
│   ├── SERVICE
│   ├── USER
│   └── SESSION
└── UTILITY_OBJECT
    ├── QUERY
    ├── REPORT
    ├── BATCH
    └── TRANSACTION
```

#### 4.2 Object-Method Matrix

| Object type | Standard methods                   | Special methods                       |
| ----------- | ---------------------------------- | ------------------------------------- |
| CUSTOMER    | CREATE, SHOW, UPDATE, DELETE, LIST | ACTIVATE, DEACTIVATE, MERGE, VALIDATE |
| INVOICE     | CREATE, SHOW, UPDATE, DELETE, LIST | SEND, MARK-PAID, CANCEL, ARCHIVE      |
| QUERY       | NEW, EXECUTE, SAVE, DELETE, LIST   | SET-SOURCE, SET-FILTER, SET-FIELDS    |
| SYSTEM      | SHOW-STATUS, SHUTDOWN, RESTART     | BACKUP, RESTORE, CONFIGURE            |

### 5  Command Processing

#### 5.1 Parse Pipeline

```
1. Tokenization
   Input: "CUST:12345:email='new@example.com'"
   Tokens: ["CUST", ":", "12345", ":", "email", "=", "'new@example.com'"]

2. Alias resolution
   Check whether "CUST" is an alias
   Result: "CUST" → "CUSTOMER"

3. Abbreviation expansion
   Expand "CUSTOMER" fully

4. Syntax validation
   Verify command matches short-form update grammar

5. Command construction
   Build CommandObject{
     Type: "CUSTOMER",
     ID: "12345",
     Method: "UPDATE",
     Params: {email: "new@example.com"}
   }

6. Security check
   Verify user has CUSTOMER:UPDATE permission

7. Routing
   Route to Customer-Service
```

#### 5.2 Error Handling

```
Error types
- SYNTAX_ERROR: invalid command syntax
- AMBIGUOUS_COMMAND: abbreviation not unique
- OBJECT_NOT_FOUND: object does not exist
- METHOD_NOT_FOUND: method unavailable for object
- PERMISSION_DENIED: insufficient privileges
- SERVICE_UNAVAILABLE: target service unreachable
- VALIDATION_ERROR: parameter validation failed
```

### 6  Alias System

#### 6.1 Alias Storage

```sql
-- Alias table
CREATE TABLE aliases (
    id UUID PRIMARY KEY,
    user_id VARCHAR(50),
    name VARCHAR(50),
    command TEXT,
    parameters TEXT[],      -- placeholder definitions
    scope VARCHAR(20),      -- 'personal', 'team', 'global'
    context VARCHAR(50),    -- optional context restriction
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP,
    last_used TIMESTAMP
);

-- Alias permissions
CREATE TABLE alias_permissions (
    alias_id UUID,
    required_role VARCHAR(50),
    FOREIGN KEY (alias_id) REFERENCES aliases(id)
);
```

#### 6.2 Alias Resolution

```go
type AliasResolver struct {
    userAliases   map[string]*Alias
    teamAliases   map[string]*Alias
    globalAliases map[string]*Alias
}

func (r *AliasResolver) Resolve(name string, context string) (*Alias, error) {
    // Priority: user > team > global
    // Respects context filters
}
```

### 7  Security Concept

#### 7.1 Authorization Model

```
Permission := Object + Method + Scope

Examples
- CUSTOMER:*:own          (only own customers)
- CUSTOMER:READ:all       (read all customers)
- CUSTOMER:DELETE:none    (no deletion)
- INVOICE:CREATE:limited  (limited by amount)
```

#### 7.2 Audit Trail

```go
type AuditEntry struct {
    Timestamp   time.Time
    UserID      string
    SessionID   string
    Command     string
    ObjectType  string
    ObjectID    string
    Method      string
    Parameters  map[string]interface{}
    Result      string
    Duration    time.Duration
    ErrorCode   string
}
```

### 8  Performance Optimization

#### 8.1 Command Cache

```go
type CommandCache struct {
    // LRU cache for parsed commands
    cache *lru.Cache

    // Pre-compiled patterns
    patterns map[string]*regexp.Regexp

    // Trie for abbreviations
    abbreviationTrie *Trie
}
```

#### 8.2 Batch Optimization

* Command batching for bulk operations
* Parallel execution where possible
* Transactional grouping

### 9  Extensibility

#### 9.1 Service Integration

New services register their objects and methods:

```protobuf
message ServiceRegistration {
    string service_id = 1;
    repeated ObjectDefinition objects = 2;
}

message ObjectDefinition {
    string object_type = 1;
    repeated MethodDefinition methods = 2;
    repeated string selectable_fields = 3;
}

message MethodDefinition {
    string name = 1;
    repeated ParameterDefinition parameters = 2;
    repeated string required_permissions = 3;
}
```

#### 9.2 Plugin System

```go
type CommandPlugin interface {
    // Plugin metadata
    GetMetadata() PluginMetadata

    // Command extensions
    GetCommands() []CommandDefinition

    // Execution
    Execute(ctx context.Context, cmd Command) (Result, error)
}
```

### 10  Implementation Roadmap

**Phase 1 – Core Implementation (4–6 weeks)**

* Basic parser and grammar
* Fundamental object types (SYSTEM, CUSTOMER, INVOICE)
* Simple methods (CREATE, SHOW, UPDATE, DELETE, LIST)
* Basic error handling

**Phase 2 – Advanced Features (4–6 weeks)**

* Full abbreviation system
* Alias system
* Batch processing
* Transactions

**Phase 3 – Enterprise Features (6–8 weeks)**

* Complex selectors and filters
* QUERY object with SQL-like syntax
* Report system
* Audit trail and security

**Phase 4 – Optimization & Polish (4 weeks)**

* Performance tuning
* Advanced error handling
* Documentation
* Tooling (syntax highlighting, auto-completion)

### 11  Sample Commands (Summary)

```bash
# Basic operations
CUSTOMER.CREATE name="Example Corp" type="B2B"
CUST:12345                                      # Short form SHOW
CUSTOMER[city="Berlin"].LIST
CUSTOMER:12345:email="new@example.com"          # Short form UPDATE

# Complex operations
QUERY.EXECUTE source="INVOICE" filter="unpaid=true AND age>30"
BATCH.EXECUTE file="month-end.tcl" mode="transaction"
REPORT[monthly-sales].GENERATE format="pdf"

# Alias definitions
ALIAS.CREATE name="uc" command="CUSTOMER.LIST filter='unpaid=true'"
ALIAS.CREATE name="morning" command=["SYSTEM.STATUS", "ORDER.LIST filter='today'"]

# System operations
SYSTEM.SHOW-STATUS
SERVICE[inventory].RESTART
TRANSACTION.EXECUTE commands=["ACCOUNT[1].WITHDRAW 1000", "ACCOUNT[2].DEPOSIT 1000"]
```
