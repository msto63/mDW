# TCOL User Guide

**Terminal Command Object Language for mDW Platform**

Version: 1.0.0  
Date: 2025-07-26  
Author: mDW Foundation Team

---

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Basic Syntax](#basic-syntax)
4. [Command Structure](#command-structure)
5. [Object Operations](#object-operations)
6. [Filtering and Selection](#filtering-and-selection)
7. [Advanced Features](#advanced-features)
8. [Business Scenarios](#business-scenarios)
9. [Best Practices](#best-practices)
10. [Troubleshooting](#troubleshooting)

---

## Introduction

TCOL (Terminal Command Object Language) is the primary interface for the Trusted Business Platform (mDW). It provides an intuitive, object-oriented command language that allows users to interact with business data and processes using natural, English-like syntax.

### Key Benefits

- **Intuitive Syntax**: Object-oriented commands that read like natural language
- **Intelligent Abbreviations**: Commands can be shortened to their unique prefixes
- **Powerful Filtering**: Advanced filtering capabilities with business logic
- **Chain Operations**: Combine multiple commands for complex workflows
- **Real-time Execution**: Immediate feedback and results
- **Audit Trail**: Complete logging of all operations for compliance

### Core Concepts

TCOL is built around the concept of **Objects** and **Methods**:
- **Objects** represent business entities (CUSTOMER, INVOICE, TASK, etc.)
- **Methods** represent actions you can perform on those objects (CREATE, UPDATE, LIST, etc.)
- **Filters** allow you to select specific objects based on criteria
- **Parameters** provide additional data for operations

---

## Getting Started

### Your First TCOL Commands

Let's start with some simple examples:

```tcol
// List all customers
CUSTOMER.LIST

// Show a specific customer
CUSTOMER:12345

// Create a new customer
CUSTOMER.CREATE name="Example Corp" type="B2B"
```

### Command Response Format

Every TCOL command returns structured data:

```json
{
  "status": "success",
  "data": {
    "customer_id": "12345",
    "name": "Example Corp",
    "type": "B2B",
    "created_date": "2024-07-26T10:30:00Z"
  },
  "execution_time": "0.045s",
  "command": "CUSTOMER.CREATE name=\"Example Corp\" type=\"B2B\""
}
```

---

## Basic Syntax

### Object.Method Pattern

The fundamental TCOL pattern is `OBJECT.METHOD`:

```tcol
CUSTOMER.CREATE     // Create a new customer
INVOICE.SEND        // Send an invoice
TASK.COMPLETE       // Complete a task
REPORT.GENERATE     // Generate a report
```

### Object Access by ID

Access specific objects using the colon notation:

```tcol
CUSTOMER:12345          // Show customer with ID 12345
INVOICE:INV-2024-001    // Show invoice by number
TASK:TSK-156            // Show task by ID
USER:john.doe           // Show user by username
```

### Field Updates

Update specific fields using the field notation:

```tcol
CUSTOMER:12345:name="Updated Name"
CUSTOMER:12345:email="new@example.com"
CUSTOMER:12345:status="active"

// Multiple field updates
CUSTOMER:12345:name="New Corp":email="contact@newcorp.com":status="active"
```

### Comments

Add comments to document your commands:

```tcol
// This is a single-line comment
CUSTOMER.CREATE name="Example Corp"

# Alternative comment style
INVOICE.SEND invoice_id="INV-001"

/* Multi-line comment
   This creates a customer
   and sends a welcome email */
CUSTOMER.CREATE name="Welcome Corp"; EMAIL.SEND template="welcome"
```

---

## Command Structure

### Basic Command Structure

```
OBJECT.METHOD [parameters] [filters] [options]
```

### Parameters

Provide data for the command using key=value pairs:

```tcol
CUSTOMER.CREATE name="Corp Name" type="B2B" city="Berlin"
INVOICE.CREATE customer_id=12345 amount=1500.00 due_date="2024-08-25"
```

### Parameter Types

- **Strings**: Use quotes for text values
  ```tcol
  name="Example Corp"
  ```

- **Numbers**: Use without quotes for numeric values
  ```tcol
  amount=1500.00
  credit_limit=50000
  ```

- **Booleans**: Use true/false
  ```tcol
  active=true
  send_notifications=false
  ```

- **Dates**: Use ISO format in quotes
  ```tcol
  due_date="2024-08-25"
  created_date="2024-07-26T10:30:00Z"
  ```

- **Arrays**: Use brackets for lists
  ```tcol
  tags=["important", "urgent"]
  product_ids=[1, 2, 3]
  ```

---

## Object Operations

### Customer Management

```tcol
// Create customers
CUSTOMER.CREATE name="Business Corp" type="B2B" city="Berlin"
CUSTOMER.CREATE name="Private Client" type="B2C" email="client@example.com"

// List and search customers
CUSTOMER.LIST
CUSTOMER.LIST limit=50 sort="name"
CUSTOMER.SEARCH query="Berlin" fields=["name", "city"]

// Update customers
CUSTOMER:12345:status="active"
CUSTOMER:12345:credit_limit=75000
CUSTOMER:12345.UPDATE status="inactive" reason="business_closure"

// Customer relationships
CUSTOMER:12345.ADD-CONTACT name="John Smith" role="CFO" email="j.smith@corp.com"
CUSTOMER:12345.ADD-ADDRESS type="billing" street="Main St 123" city="Berlin"
```

### Invoice Management

```tcol
// Create invoices
INVOICE.CREATE customer_id=12345 amount=2500.00 due_date="2024-08-25"
INVOICE.CREATE customer_id=12346 items=["SRV-001", "SRV-002"] tax_rate=0.19

// Invoice operations
INVOICE:INV-001.SEND method="email" copy_accounting=true
INVOICE:INV-001.PAY amount=2500.00 method="bank_transfer"
INVOICE:INV-001.CANCEL reason="customer_request"

// Invoice line items
INVOICE:INV-001.ADD-ITEM product="CONSULTING" quantity=10 unit_price=150.00
INVOICE:INV-001.UPDATE-ITEM item_id=1 quantity=12 unit_price=140.00

// Payment tracking
PAYMENT.CREATE invoice_id="INV-001" amount=2500.00 method="credit_card"
PAYMENT:PAY-001:status="processed" reference="TXN-789456"
```

### Task and Project Management

```tcol
// Create tasks
TASK.CREATE title="Implement Feature X" priority=2 due_date="2024-08-15"
TASK.CREATE title="Customer Onboarding" assignee="sales.team" customer_id=12345

// Task management
TASK:TSK-156.ASSIGN user_id="john.doe" notes="Urgent priority"
TASK:TSK-156.UPDATE status="in_progress" completion=25
TASK:TSK-156.COMPLETE notes="Feature implemented successfully"

// Project operations
PROJECT.CREATE name="Q3 Initiative" budget=100000 start_date="2024-07-01"
PROJECT:PRJ-001.ADD-MEMBER user_id="maria.smith" role="Project Manager"
PROJECT:PRJ-001.ADD-MILESTONE name="Phase 1 Complete" date="2024-08-31"
```

---

## Filtering and Selection

### Basic Filters

Use brackets to filter objects based on criteria:

```tcol
// Simple filters
CUSTOMER[status="active"].LIST
CUSTOMER[city="Berlin"].LIST
CUSTOMER[type="B2B"].LIST

// Numeric filters
INVOICE[amount>1000].LIST
CUSTOMER[credit_limit>=50000].LIST
TASK[priority>=3].LIST
```

### Multiple Conditions

Combine multiple filter conditions:

```tcol
// AND conditions (comma-separated)
CUSTOMER[status="active",type="B2B"].LIST
INVOICE[status="unpaid",amount>500].LIST

// OR conditions
CUSTOMER[city="Berlin" OR city="Munich"].LIST
TASK[priority=1 OR priority=2].LIST

// Complex conditions
CUSTOMER[status="active" AND (type="B2B" OR credit_limit>10000)].LIST
```

### Comparison Operators

- `=` or `==`: Equals
- `!=` or `<>`: Not equals
- `>`: Greater than
- `>=`: Greater than or equal
- `<`: Less than
- `<=`: Less than or equal
- `LIKE`: String pattern matching
- `IN`: Value in list
- `BETWEEN`: Value in range

```tcol
// String operations
CUSTOMER[name LIKE "Corp*"].LIST
CUSTOMER[city IN ["Berlin", "Munich", "Hamburg"]].LIST

// Date ranges
INVOICE[created_date BETWEEN "2024-01-01" AND "2024-12-31"].LIST
TASK[due_date<="2024-07-31"].LIST

// Null checks
CUSTOMER[email IS NOT NULL].LIST
TASK[assignee IS NULL].LIST
```

### Advanced Filtering

```tcol
// Nested object filtering
CUSTOMER[orders.total_amount>5000].LIST
INVOICE[customer.type="B2B"].LIST

// Date calculations
INVOICE[age>30].LIST                    // Invoices older than 30 days
CUSTOMER[last_order<"30_days_ago"].LIST // Inactive customers

// Array filtering
CUSTOMER[tags CONTAINS "vip"].LIST
PRODUCT[categories CONTAINS_ANY ["electronics", "computers"]].LIST
```

---

## Advanced Features

### Command Abbreviations

TCOL supports intelligent abbreviations. You can shorten commands to their unique prefixes:

```tcol
// Full commands
CUSTOMER.CREATE name="Corp"
CUSTOMER.LIST
CUSTOMER.UPDATE

// Abbreviated versions
CUST.CR name="Corp"      // CUSTOMER.CREATE
CUST.L                   // CUSTOMER.LIST  
CUST.U                   // CUSTOMER.UPDATE

// Object abbreviations
C:12345                  // CUSTOMER:12345
I:INV-001               // INVOICE:INV-001
T:TSK-156               // TASK:TSK-156
```

### Command Chaining

Chain multiple commands together:

```tcol
// Sequential execution (semicolon)
CUSTOMER.CREATE name="New Corp"; CUSTOMER[name="New Corp"].LIST

// Conditional execution
CUSTOMER:12345 && CUSTOMER:12345:status="active"  // Execute second if first succeeds
CUSTOMER:99999 || CUSTOMER.CREATE id=99999        // Execute second if first fails

// Piped operations
CUSTOMER[city="Berlin"].LIST | EXPORT.CSV filename="berlin-customers.csv"
INVOICE[status="unpaid"].LIST | EMAIL.SEND template="reminder"
```

### Aliases

Create shortcuts for frequently used commands:

```tcol
// Create aliases
ALIAS.CREATE name="ac" command="CUSTOMER[status='active'].LIST"
ALIAS.CREATE name="up" command="INVOICE[status='unpaid'].LIST"
ALIAS.CREATE name="newinv" command="INVOICE.CREATE customer_id=$1 amount=$2"

// Use aliases
ac                          // Lists active customers
up                          // Lists unpaid invoices  
newinv 12345 1500.00       // Creates invoice with parameters
```

### Variables and Context

Use variables to store and reuse values:

```tcol
// Set variables
SET customer_id = 12345
SET invoice_amount = 2500.00

// Use variables
CUSTOMER:$customer_id
INVOICE.CREATE customer_id=$customer_id amount=$invoice_amount

// System variables
USER.ID                     // Current user ID
SESSION.TIMESTAMP           // Current session timestamp
REQUEST.IP                  // Request IP address
```

### Batch Operations

Process multiple items efficiently:

```tcol
// Batch creation
CUSTOMER.BULK_CREATE file="customers.csv" batch_size=100

// Batch updates
CUSTOMER[city="Berlin"].BULK_UPDATE status="verified" batch_size=50

// Batch processing with progress
BATCH.PROCESS items=invoice_list operation="INVOICE.SEND" progress=true
```

### Conditional Logic

Use IF-THEN-ELSE for conditional execution:

```tcol
// Simple conditions
IF CUSTOMER:12345.EXISTS THEN {
    CUSTOMER:12345:status="active"
} ELSE {
    CUSTOMER.CREATE id=12345 name="New Customer"
}

// Complex conditions
IF INVOICE[status="overdue"].COUNT > 10 THEN {
    ALERT.CREATE message="High overdue count" severity="high"
    TASK.CREATE title="Review overdue invoices" assignee="collections.team"
} ELSE {
    LOG.INFO "Overdue invoices within normal range"
}
```

### Error Handling

Handle errors gracefully:

```tcol
// Try-catch blocks
TRY {
    CUSTOMER:12345:email="invalid-email"
} CATCH ValidationError {
    LOG.ERROR "Email validation failed"
    CUSTOMER:12345.NOTIFY "Please provide valid email"
} CATCH SystemError {
    ALERT.ESCALATE "System error during update"
}

// Error recovery
CUSTOMER.UPDATE customer_id=12345 email="new@example.com" 
ON_ERROR RETRY count=3 delay=1000
ON_FINAL_ERROR LOG.ERROR "Failed to update customer email"
```

---

## Business Scenarios

### Customer Onboarding

```tcol
// Complete customer onboarding workflow
CUSTOMER.CREATE name="TechStart GmbH" type="B2B" industry="Technology"
SET new_customer_id = RESULT.customer_id

// Set up customer details
CUSTOMER:$new_customer_id:credit_limit=50000 :payment_terms="NET30"
CONTACT.CREATE customer_id=$new_customer_id name="John Smith" role="CFO"

// Create welcome tasks
TASK.CREATE title="Welcome Package" customer_id=$new_customer_id assignee="sales.team"
TASK.CREATE title="Technical Setup" customer_id=$new_customer_id assignee="tech.team"

// Send welcome communications
EMAIL.SEND customer_id=$new_customer_id template="welcome_b2b"
DOCUMENT.GENERATE customer_id=$new_customer_id template="service_agreement"
```

### Monthly Invoice Processing

```tcol
// Monthly invoice generation and processing
SET current_month = "2024-07"

// Generate invoices for all recurring services
ORDER[type="recurring",billing_month=$current_month].GENERATE_INVOICES

// Process and send invoices
INVOICE[status="draft",created_date>=$current_month].FINALIZE batch_size=100
INVOICE[status="finalized"].SEND method="email" parallel=true

// Set up payment tracking
INVOICE[status="sent"].CREATE_PAYMENT_REMINDERS schedule="7,14,30"

// Generate monthly reports
REPORT.GENERATE type="invoice_summary" period=$current_month
REPORT.GENERATE type="aging_report" as_of=$current_month
```

### Sales Pipeline Management

```tcol
// Daily sales pipeline review
SET today = TODAY()

// Update opportunity stages based on activities
OPPORTUNITY[last_activity<"7_days_ago"].UPDATE status="stale"
OPPORTUNITY[close_date<$today,status!="closed"].UPDATE status="overdue"

// Generate follow-up tasks
OPPORTUNITY[status="stale"].CREATE_TASK title="Re-engage prospect" assignee=owner
OPPORTUNITY[status="overdue"].CREATE_TASK title="Update close date" priority=2

// Sales performance metrics
METRICS.CALCULATE period="current_quarter" metrics=["pipeline_value","conversion_rate","avg_deal_size"]
FORECAST.UPDATE period="current_quarter" confidence=80
```

---

## Best Practices

### Command Structure

1. **Use Clear Object Names**: Prefer full object names over abbreviations for readability
2. **Consistent Parameter Naming**: Use consistent parameter names across commands
3. **Meaningful Comments**: Document complex command sequences
4. **Error Handling**: Always include error handling for critical operations

### Performance Optimization

1. **Use Filters**: Apply filters to reduce data processing
2. **Batch Operations**: Use bulk operations for multiple items
3. **Limit Results**: Use limit parameters for large datasets
4. **Index-Friendly Filters**: Filter on indexed fields when possible

```tcol
// Good: Filtered and limited
CUSTOMER[status="active"].LIST limit=100 sort="name"

// Less efficient: No filtering
CUSTOMER.LIST
```

### Security Best Practices

1. **Parameter Validation**: Validate all input parameters
2. **Access Control**: Ensure proper permissions for sensitive operations
3. **Audit Trails**: Include audit information for compliance
4. **Data Masking**: Mask sensitive data in logs and outputs

```tcol
// Include audit context
CUSTOMER:12345:credit_limit=100000 audit_reason="credit_review" approved_by="manager"

// Validate sensitive operations
VALIDATE.PERMISSIONS action="CUSTOMER.DELETE" resource_id=12345
```

### Code Organization

1. **Use Scripts**: Create reusable TCOL scripts for complex workflows
2. **Variable Management**: Use meaningful variable names
3. **Modular Approach**: Break complex operations into smaller commands
4. **Documentation**: Document business rules and processes

---

## Troubleshooting

### Common Error Types

#### Syntax Errors
```tcol
// Error: Missing quotes around string
CUSTOMER.CREATE name=Example Corp  ❌

// Correct: String in quotes
CUSTOMER.CREATE name="Example Corp"  ✅
```

#### Validation Errors
```tcol
// Error: Invalid email format
CUSTOMER.CREATE email="invalid-email"  ❌

// Correct: Valid email format
CUSTOMER.CREATE email="user@example.com"  ✅
```

#### Permission Errors
```tcol
// Error: Insufficient permissions
CUSTOMER:12345.DELETE  ❌

// Check permissions first
AUTHORIZE action="CUSTOMER.DELETE" resource_id=12345
```

### Performance Issues

#### Slow Queries
```tcol
// Problematic: No filtering
CUSTOMER.LIST  // Returns all customers

// Better: Use filters
CUSTOMER[status="active"].LIST limit=100
```

#### Large Batch Operations
```tcol
// Problematic: Too large batch
CUSTOMER.BULK_CREATE file="million_customers.csv"

// Better: Smaller batches
CUSTOMER.BULK_CREATE file="million_customers.csv" batch_size=1000
```

### Debugging Commands

```tcol
// Enable debug logging
LOG.LEVEL debug

// Explain query execution
EXPLAIN CUSTOMER[city="Berlin"].LIST

// Performance analysis
PROFILE.START
COMPLEX.OPERATION
PROFILE.END save_results=true

// Check system status
SYSTEM.STATUS
HEALTH.CHECK components=["database","cache","messaging"]
```

### Error Recovery

```tcol
// Automatic retry for transient errors
CUSTOMER.UPDATE customer_id=12345 status="active"
RETRY on_error="TransientError" max_attempts=3 delay=1000

// Graceful degradation
TRY {
    CUSTOMER.ADVANCED_ANALYTICS customer_id=12345
} CATCH ServiceUnavailableError {
    CUSTOMER.BASIC_STATS customer_id=12345
    LOG.WARN "Advanced analytics unavailable, using basic stats"
}
```

---

## Getting Help

### Built-in Help System

```tcol
// General help
HELP

// Object-specific help
HELP CUSTOMER
HELP INVOICE

// Command-specific help
HELP CUSTOMER.CREATE
HELP INVOICE.SEND

// Example commands
EXAMPLES CUSTOMER
EXAMPLES "invoice processing"
```

### Support Resources

- **User Manual**: Complete documentation with examples
- **Video Tutorials**: Step-by-step video guides
- **Community Forum**: User community and Q&A
- **Support Tickets**: Direct technical support
- **Training Sessions**: Live training and workshops

### Contact Information

- **Technical Support**: support@mdw-platform.local
- **Documentation**: docs@mdw-platform.local
- **Training**: training@mdw-platform.local

---

*This guide covers the essential TCOL features for business users. For advanced technical features and integration details, see the TCOL Developer Guide.*