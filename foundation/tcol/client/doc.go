// File: doc.go
// Title: TCOL Service Client Package Documentation
// Description: Implements client interfaces for communicating with mDW
//              microservices. Provides gRPC-based communication, connection
//              management, and service discovery for TCOL command execution.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial client implementation

/*
Package client provides service communication capabilities for TCOL.

This package implements client interfaces for communicating with mDW
microservices during TCOL command execution. It includes:

  • gRPC client implementations for service communication
  • Connection pooling and management
  • Service discovery and health checking
  • Request/response serialization
  • Circuit breaker patterns for resilience

The client integrates with the executor to provide reliable communication
with the distributed mDW microservice architecture.
*/
package client