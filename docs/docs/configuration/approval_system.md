# Approval System

The Distributed Knowledge approval system allows you to control which queries are automatically processed and which require manual approval. This document explains how to configure and use this system effectively.

## Overview

The approval system consists of:

1. **Automatic Approval Rules**: Conditions that determine which queries to automatically accept
2. **Manual Review Queue**: Queries that don't meet automatic approval criteria await manual decision
3. **MCP Tools**: Functions to manage approvals, rejections, and rule configuration

## Automatic Approval Configuration

Automatic approval rules are defined in a JSON file specified with the `-automaticApproval` parameter:

```bash
./dk -automaticApproval="./config/automatic_approval.json"
```

### Rule Format

The automatic approval file contains an array of rule strings:

```json
[
  "Accept all questions about public information",
  "Allow queries from trusted_user",
  "Permit questions related to scientific topics",
  "Reject questions about personal finances"
]
```

Each rule is evaluated against incoming queries to determine if they should be automatically approved, rejected, or held for manual review.

## Managing Approval Rules with MCP Tools

The MCP server provides tools for managing approval rules at runtime:

### Adding Rules

Use the `cqAddAutoApprovalCondition` tool to add new rules:

```json
{
  "name": "cqAddAutoApprovalCondition",
  "parameters": {
    "sentence": "Accept questions about programming languages"
  }
}
```

### Removing Rules

Use the `cqRemoveAutoApprovalCondition` tool to remove existing rules:

```json
{
  "name": "cqRemoveAutoApprovalCondition",
  "parameters": {
    "condition": "Accept questions about programming languages"
  }
}
```

### Listing Rules

Use the `cqListAutoApprovalConditions` tool to view all current rules:

```json
{
  "name": "cqListAutoApprovalConditions"
}
```

## Query Review Process

Queries that don't match automatic approval rules are placed in a pending state:

1. **Incoming Query**: The system receives a query from the network
2. **Rule Evaluation**: The query is checked against automatic approval rules
3. **Automatic Disposition**: If matched, the query is automatically approved or rejected
4. **Pending Queue**: If no matching rule, the query is placed in the pending queue
5. **Manual Review**: User reviews pending queries and decides to accept or reject

## Manual Review Using MCP Tools

### Listing Pending Queries

View queries that require manual review:

```json
{
  "name": "cqListRequestedQueries",
  "parameters": {
    "status": "pending"
  }
}
```

### Accepting Queries

Approve a pending query to generate and send a response:

```json
{
  "name": "cqAcceptQuery",
  "parameters": {
    "id": "qry-123"
  }
}
```

### Rejecting Queries

Reject a query that doesn't warrant a response:

```json
{
  "name": "cqRejectQuery",
  "parameters": {
    "id": "qry-123"
  }
}
```

## Rule Evaluation Process

When a query is received, the system:

1. Extracts relevant features (sender, topic, question content)
2. Compares these features against each rule in the approval configuration
3. If a matching rule is found, applies the corresponding action (accept/reject)
4. If no matching rule is found, marks the query as pending

## Effective Rule Writing

Guidelines for writing effective approval rules:

### Rule Structure

Rules should follow a consistent structure:

- **Action Verb**: Begin with "Accept", "Allow", "Permit", or "Reject"
- **Condition**: Specify the condition for the action
- **Specificity**: Be as specific as possible to avoid ambiguity

### Rule Examples

Good examples of specific rules:

```
"Accept questions about public scientific research from academic institutions"
"Allow queries related to programming if they don't request sensitive code"
"Reject any question containing requests for personal information"
"Permit questions from user_id=trusted_researcher about quantum physics"
```

Avoid vague rules:

```
"Accept good questions"  // Too vague
"Reject bad requests"    // Undefined criteria
```

### Rule Priority

Rules are evaluated in order, with earlier rules taking precedence. Organize your rules with this in mind:

1. Place specific rules before general ones
2. Put rejection rules before acceptance rules if in doubt
3. End with a default rule if you want a catch-all behavior

## Advanced Configuration

### Customizing the Approval System

The approval system can be customized with additional parameters:

```bash
./dk -automaticApproval="./config/automatic_approval.json" \
     -approval_mode="strict" \
     -default_action="reject" \
     -max_pending_time=86400
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-approval_mode` | Mode of operation (strict, lenient, balanced) | balanced |
| `-default_action` | Default action if no rule matches (accept, reject, pending) | pending |
| `-max_pending_time` | Maximum time in seconds a query can remain pending | 604800 (7 days) |

### Approval Modes

The system supports different approval modes:

- **Strict**: Requires explicit approval rules to match, defaults to rejection
- **Balanced**: Allows specific types of queries automatically, others require approval
- **Lenient**: Accepts most queries by default, only rejects explicit matches

## Integration with External Systems

The approval system can integrate with external validation services:

```bash
./dk -external_approval_url="https://your-validator-service.com/api/validate" \
     -external_approval_token="your-api-token" \
     -external_approval_timeout=5
```

This configuration forwards queries to an external validation service for additional checks.

## Best Practices

1. **Start Conservative**: Begin with a strict approval policy and relax it as needed
2. **Regular Maintenance**: Review and update rules as your needs evolve
3. **Monitor Patterns**: Look for common query patterns that should be covered by rules
4. **Specific User Rules**: Create rules for specific trusted users who should have more access
5. **Topic-Based Rules**: Organize rules around knowledge domains or topics
6. **Test New Rules**: Verify new rules work as expected before deploying them
7. **Document Rules**: Maintain documentation of your rule set and the reasoning behind it

## Common Issues and Solutions

| Issue | Solution |
|-------|----------|
| Too many pending queries | Add more specific automatic approval rules |
| Inappropriate automatic approvals | Make rules more specific or add rejection rules |
| Rules not matching as expected | Review rule phrasing and order of evaluation |
| Overwhelmed by manual reviews | Implement more granular automatic rules |
