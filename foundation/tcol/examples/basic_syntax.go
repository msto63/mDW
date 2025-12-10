// File: basic_syntax.go
// Package: examples
// Title: TCOL Basic Syntax Examples
// Description: Demonstrates fundamental TCOL command syntax patterns,
//              object-method operations, and basic command structures
//              for the Terminal Command Object Language.
// Author: msto63 with Claude Opus 4.0
// Version: v0.1.0
// Created: 2025-07-26
// Modified: 2025-07-26

package examples

import (
	"fmt"
	"strings"
)

// TCOLBasicSyntaxDemo demonstrates fundamental TCOL command patterns
type TCOLBasicSyntaxDemo struct {
	commands []string
}

// NewBasicSyntaxDemo creates a new demonstration instance
func NewBasicSyntaxDemo() *TCOLBasicSyntaxDemo {
	return &TCOLBasicSyntaxDemo{
		commands: make([]string, 0),
	}
}

// BasicObjectMethodSyntax demonstrates the core Object.Method pattern
func (demo *TCOLBasicSyntaxDemo) BasicObjectMethodSyntax() []string {
	examples := []string{
		// Basic object operations
		"CUSTOMER.CREATE",
		"CUSTOMER.LIST",
		"CUSTOMER.UPDATE",
		"CUSTOMER.DELETE",
		
		// Invoice operations
		"INVOICE.CREATE",
		"INVOICE.SEND",
		"INVOICE.PAY",
		"INVOICE.CANCEL",
		
		// Task operations
		"TASK.CREATE",
		"TASK.ASSIGN",
		"TASK.COMPLETE",
		"TASK.ARCHIVE",
		
		// System operations
		"SYSTEM.STATUS",
		"SYSTEM.BACKUP",
		"SYSTEM.RESTART",
		"CONFIG.RELOAD",
	}
	
	demo.logExamples("Basic Object.Method Syntax", examples)
	return examples
}

// ObjectIdentifierAccess demonstrates direct object access by ID
func (demo *TCOLBasicSyntaxDemo) ObjectIdentifierAccess() []string {
	examples := []string{
		// Direct object access by ID
		"CUSTOMER:12345",          // Show customer with ID 12345
		"INVOICE:INV-2024-001",    // Show invoice by number
		"TASK:TSK-2024-0156",      // Show task by ID
		"USER:john.doe",           // Show user by username
		
		// Nested object access
		"CUSTOMER:12345:CONTACT",  // Show customer's contact info
		"INVOICE:INV-001:ITEMS",   // Show invoice items
		"TASK:TSK-156:COMMENTS",   // Show task comments
		"PROJECT:PRJ-001:TEAM",    // Show project team
	}
	
	demo.logExamples("Object Identifier Access", examples)
	return examples
}

// FieldUpdateOperations demonstrates field update syntax
func (demo *TCOLBasicSyntaxDemo) FieldUpdateOperations() []string {
	examples := []string{
		// Single field updates
		"CUSTOMER:12345:name=\"Updated Corp\"",
		"CUSTOMER:12345:email=\"new@example.com\"",
		"CUSTOMER:12345:status=\"active\"",
		
		// Multiple field updates
		"CUSTOMER:12345:name=\"New Corp\":email=\"contact@newcorp.com\"",
		"INVOICE:INV-001:status=\"paid\":paid_date=\"2024-07-26\"",
		"TASK:TSK-156:status=\"completed\":completion_date=\"2024-07-26\"",
		
		// Numeric field updates
		"INVOICE:INV-001:amount=1250.00",
		"CUSTOMER:12345:credit_limit=50000.00",
		"TASK:TSK-156:priority=3",
	}
	
	demo.logExamples("Field Update Operations", examples)
	return examples
}

// FilteringSyntax demonstrates TCOL filtering capabilities
func (demo *TCOLBasicSyntaxDemo) FilteringSyntax() []string {
	examples := []string{
		// Simple filters
		"CUSTOMER[status=\"active\"].LIST",
		"CUSTOMER[city=\"Berlin\"].LIST",
		"CUSTOMER[type=\"B2B\"].LIST",
		
		// Multiple conditions
		"CUSTOMER[status=\"active\",type=\"B2B\"].LIST",
		"INVOICE[status=\"unpaid\",amount>1000].LIST",
		"TASK[priority>=3,status!=\"completed\"].LIST",
		
		// Date range filters
		"INVOICE[created_date>=\"2024-01-01\",created_date<=\"2024-12-31\"].LIST",
		"CUSTOMER[last_contact>=\"2024-07-01\"].LIST",
		"TASK[due_date<=\"2024-07-31\",status!=\"completed\"].LIST",
		
		// Complex boolean expressions
		"CUSTOMER[status=\"active\" AND (type=\"B2B\" OR credit_limit>10000)].LIST",
		"INVOICE[status=\"unpaid\" AND age>30 AND amount>500].SEND-REMINDER",
	}
	
	demo.logExamples("Filtering Syntax", examples)
	return examples
}

// ParameterizedCommands demonstrates commands with parameters
func (demo *TCOLBasicSyntaxDemo) ParameterizedCommands() []string {
	examples := []string{
		// Customer creation with parameters
		"CUSTOMER.CREATE name=\"Example Corp\" type=\"B2B\" city=\"Berlin\"",
		"CUSTOMER.CREATE name=\"Small Shop\" type=\"B2C\" email=\"info@shop.com\"",
		
		// Invoice creation
		"INVOICE.CREATE customer_id=12345 amount=1500.00 due_date=\"2024-08-25\"",
		"INVOICE.CREATE customer_id=12346 items=\"item1,item2\" tax_rate=0.19",
		
		// Task creation and assignment
		"TASK.CREATE title=\"Implement Feature X\" priority=2 due_date=\"2024-08-15\"",
		"TASK.ASSIGN task_id=TSK-156 user_id=\"john.doe\" notes=\"Urgent priority\"",
		
		// Search and query operations
		"QUERY.EXECUTE source=\"CUSTOMER\" filter=\"city='Berlin'\" limit=50",
		"SEARCH.FIND text=\"important document\" scope=\"all\" user_id=\"john.doe\"",
		
		// Batch operations
		"BATCH.EXECUTE file=\"monthly-reports.tcl\" mode=\"parallel\"",
		"BATCH.CREATE name=\"end-of-month\" commands=\"INVOICE.SEND,REPORT.GENERATE\"",
	}
	
	demo.logExamples("Parameterized Commands", examples)
	return examples
}

// AbbreviationExamples demonstrates command abbreviations
func (demo *TCOLBasicSyntaxDemo) AbbreviationExamples() []string {
	examples := []string{
		// Basic abbreviations (minimum unique characters)
		"CUST.CR",        // CUSTOMER.CREATE
		"CUST.L",         // CUSTOMER.LIST
		"CUST.U",         // CUSTOMER.UPDATE
		"CUST.D",         // CUSTOMER.DELETE
		
		"INV.CR",         // INVOICE.CREATE
		"INV.S",          // INVOICE.SEND
		"INV.P",          // INVOICE.PAY
		"INV.C",          // INVOICE.CANCEL
		
		"TSK.CR",         // TASK.CREATE
		"TSK.AS",         // TASK.ASSIGN
		"TSK.CO",         // TASK.COMPLETE
		
		// Longer abbreviations for clarity
		"CUST.CREATE",    // CUSTOMER.CREATE
		"INV.SEND",       // INVOICE.SEND
		"TSK.ASSIGN",     // TASK.ASSIGN
		
		// Object abbreviations
		"C:12345",        // CUSTOMER:12345
		"I:INV-001",      // INVOICE:INV-001
		"T:TSK-156",      // TASK:TSK-156
		
		// Combined abbreviations with filters
		"CUST[st=\"active\"].L",           // CUSTOMER[status="active"].LIST
		"INV[st=\"unpaid\"].S-REM",        // INVOICE[status="unpaid"].SEND-REMINDER
	}
	
	demo.logExamples("Command Abbreviations", examples)
	return examples
}

// ChainedCommands demonstrates command chaining and pipes
func (demo *TCOLBasicSyntaxDemo) ChainedCommands() []string {
	examples := []string{
		// Sequential command execution
		"CUSTOMER.CREATE name=\"New Corp\"; CUSTOMER[name=\"New Corp\"].LIST",
		"INVOICE.CREATE customer_id=12345; INVOICE.SEND invoice_id=last_created",
		
		// Conditional execution
		"CUSTOMER:12345 && CUSTOMER:12345:status=\"active\"",
		"INVOICE:INV-001 || INVOICE.CREATE customer_id=12345",
		
		// Piped operations (output of first becomes input of second)
		"CUSTOMER[city=\"Berlin\"].LIST | EXPORT.CSV filename=\"berlin-customers.csv\"",
		"INVOICE[status=\"unpaid\"].LIST | EMAIL.SEND template=\"reminder\"",
		"TASK[assignee=\"john.doe\"].LIST | REPORT.GENERATE format=\"pdf\"",
		
		// Complex chains
		"CUSTOMER.CREATE name=\"Test Corp\"; C:last_created:email=\"test@corp.com\"; C:last_created",
		"INV[status=\"overdue\"].LIST | EMAIL.SEND template=\"urgent\" | LOG.AUDIT action=\"reminder_sent\"",
	}
	
	demo.logExamples("Chained Commands", examples)
	return examples
}

// CommentAndDocumentation demonstrates TCOL comments and inline documentation
func (demo *TCOLBasicSyntaxDemo) CommentAndDocumentation() []string {
	examples := []string{
		// Single line comments
		"// Create a new business customer",
		"CUSTOMER.CREATE name=\"Business Corp\" type=\"B2B\"",
		"",
		"// Send reminder to all overdue invoices",
		"INVOICE[status=\"overdue\"].SEND-REMINDER",
		"",
		"# Alternative comment style",
		"TASK.CREATE title=\"Important Task\" priority=1",
		"",
		"/* Multi-line comment",
		"   This command creates a customer",
		"   and immediately shows the details */",
		"CUSTOMER.CREATE name=\"Example\"; CUSTOMER:last_created",
		"",
		// Inline documentation
		"CUSTOMER.CREATE name=\"Corp\" type=\"B2B\"  // Creates B2B customer",
		"INVOICE[amount>1000].LIST  # List high-value invoices",
	}
	
	demo.logExamples("Comments and Documentation", examples)
	return examples
}

// ValidationAndErrorHandling demonstrates error handling patterns
func (demo *TCOLBasicSyntaxDemo) ValidationAndErrorHandling() []string {
	examples := []string{
		// Input validation
		"CUSTOMER.CREATE name=required email=optional type=enum[B2B,B2C]",
		"INVOICE.CREATE customer_id=exists amount=positive due_date=future",
		
		// Error handling with try-catch equivalent
		"TRY { CUSTOMER:12345:email=\"invalid-email\" } CATCH { LOG.ERROR \"Invalid email format\" }",
		"TRY { INVOICE.SEND invoice_id=nonexistent } CATCH { CUSTOMER.NOTIFY \"Invoice not found\" }",
		
		// Conditional execution based on existence
		"IF CUSTOMER:12345 THEN { CUSTOMER:12345:status=\"active\" } ELSE { CUSTOMER.CREATE id=12345 }",
		"IF INVOICE[status=\"draft\"] THEN { INVOICE.FINALIZE } ELSE { LOG.INFO \"No drafts to finalize\" }",
		
		// Validation commands
		"VALIDATE.EMAIL email=\"test@example.com\"",
		"VALIDATE.CUSTOMER customer_id=12345 required_fields=\"name,email,type\"",
		"VALIDATE.INVOICE invoice_id=INV-001 business_rules=\"due_date_future,amount_positive\"",
	}
	
	demo.logExamples("Validation and Error Handling", examples)
	return examples
}

// logExamples logs examples with proper formatting
func (demo *TCOLBasicSyntaxDemo) logExamples(title string, examples []string) {
	fmt.Printf("\n=== %s ===\n", title)
	for i, example := range examples {
		if strings.TrimSpace(example) == "" {
			fmt.Println()
		} else {
			fmt.Printf("%2d. %s\n", i+1, example)
		}
	}
	demo.commands = append(demo.commands, examples...)
}

// GetAllCommands returns all demonstration commands
func (demo *TCOLBasicSyntaxDemo) GetAllCommands() []string {
	return demo.commands
}

// RunAllDemonstrations executes all syntax demonstrations
func (demo *TCOLBasicSyntaxDemo) RunAllDemonstrations() {
	fmt.Println("TCOL Basic Syntax Demonstration")
	fmt.Println("===============================")
	
	demo.BasicObjectMethodSyntax()
	demo.ObjectIdentifierAccess()
	demo.FieldUpdateOperations()
	demo.FilteringSyntax()
	demo.ParameterizedCommands()
	demo.AbbreviationExamples()
	demo.ChainedCommands()
	demo.CommentAndDocumentation()
	demo.ValidationAndErrorHandling()
	
	fmt.Printf("\nTotal examples demonstrated: %d\n", len(demo.commands))
}