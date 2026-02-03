# AgentShield Architecture & Development Guide

## Overview

AgentShield is a local-first security gateway that sits between AI agents and high-risk tools, enforcing deterministic policies to prevent prompt-injection-driven damage, data exfiltration, and destructive actions.

## Architecture Flow Diagrams

### 1. High-Level System Architecture

```mermaid
graph TB
    subgraph "AI Agent"
        A[AI Agent/LLM]
    end
    
    subgraph "AgentShield Gateway"
        B[CLI Entry Point<br/>cmd/agentshield/main.go]
        C[Command Parser<br/>internal/cli/root.go]
        D[Run Command Handler<br/>internal/cli/run.go]
    end
    
    subgraph "Core Processing"
        E[Config Loader<br/>internal/config/]
        F[Command Normalizer<br/>internal/normalize/]
        G[Policy Engine<br/>internal/policy/]
        H[Audit Logger<br/>internal/logger/]
    end
    
    subgraph "Decision Enforcement"
        I[Block Handler]
        J[Approval Handler<br/>internal/approval/]
        K[Sandbox Handler<br/>internal/sandbox/]
        L[Direct Execution]
    end
    
    subgraph "System Resources"
        M[File System]
        N[Terminal/Shell]
        O[Network/External Tools]
    end
    
    A --> B
    B --> C
    C --> D
    D --> E
    D --> F
    D --> G
    D --> H
    G --> I
    G --> J
    G --> K
    G --> L
    I --> M
    J --> N
    K --> M
    L --> N
    K --> O
```

### 2. Command Execution Flow

```mermaid
flowchart TD
    Start([agentshield run -- command]) --> ParseArgs[Parse Command Arguments]
    ParseArgs --> LoadConfig[Load Configuration]
    LoadConfig --> InitLogger[Initialize Audit Logger]
    InitLogger --> NormalizeCmd[Normalize Command]
    NormalizeCmd --> LoadPolicy[Load Policy Rules]
    LoadPolicy --> CreateEngine[Create Policy Engine]
    CreateEngine --> Evaluate{Evaluate Policy}
    
    Evaluate -->|Block| BlockFlow[Block Command]
    Evaluate -->|Require Approval| ApprovalFlow[Request User Approval]
    Evaluate -->|Sandbox| SandboxFlow[Run in Sandbox]
    Evaluate -->|Allow| DirectFlow[Execute Directly]
    
    BlockFlow --> LogBlock[Log Block Event]
    ApprovalFlow --> UserDecision{User Decision?}
    SandboxFlow --> RunSandbox[Run in Sandbox]
    DirectFlow --> ExecuteCmd[Execute Command]
    
    UserDecision -->|Approve| ExecuteCmd
    UserDecision -->|Deny| LogDeny[Log Deny Event]
    
    RunSandbox --> ShowResults[Show Sandbox Results]
    ShowResults --> ApplyChanges{Apply Changes?}
    ApplyChanges -->|Yes| ApplyToReal[Apply to Real System]
    ApplyChanges -->|No| LogNoApply[Log No Apply]
    
    ExecuteCmd --> LogSuccess[Log Execution]
    ApplyToReal --> LogSuccess
    
    LogBlock --> End([Exit with Error])
    LogDeny --> End
    LogNoApply --> End
    LogSuccess --> EndDone([Exit Success])
```

### 3. Policy Engine Decision Flow

```mermaid
flowchart TD
    StartEval[Start Policy Evaluation] --> CheckProtected{Check Protected Paths}
    CheckProtected -->|Protected| BlockDecision[Decision: Block]
    CheckProtected -->|Not Protected| IterateRules[Iterate Through Rules]
    
    IterateRules --> MatchRule{Rule Matches Command?}
    MatchRule -->|No| NextRule{More Rules?}
    MatchRule -->|Yes| ApplyRule[Apply Rule Decision]
    
    NextRule -->|Yes| IterateRules
    NextRule -->|No| UseDefault[Use Default Decision]
    
    BlockDecision --> BuildResult[Build Evaluation Result]
    ApplyRule --> BuildResult
    UseDefault --> BuildResult
    
    BuildResult --> ReturnResult[Return Evaluation Result]
    
    subgraph "Rule Matching Logic"
        ExactMatch{Command Exact Match?}
        PrefixMatch{Command Prefix Match?}
        RegexMatch{Command Regex Match?}
        
        ExactMatch -->|No| PrefixMatch
        ExactMatch -->|Yes| RuleMatched[Rule Matched]
        PrefixMatch -->|No| RegexMatch
        PrefixMatch -->|Yes| RuleMatched
        RegexMatch -->|No| NoMatch[No Match]
        RegexMatch -->|Yes| RuleMatched
    end
```

### 4. Internal Package Dependencies

```mermaid
graph LR
    subgraph "Entry Point"
        A[main.go]
    end
    
    subgraph "CLI Layer"
        B[cli/root.go]
        C[cli/run.go]
        D[cli/version.go]
    end
    
    subgraph "Core Services"
        E[config/]
        F[logger/]
        G[normalize/]
        H[policy/]
        I[approval/]
        J[sandbox/]
        K[redact/]
    end
    
    A --> B
    B --> C
    C --> E
    C --> F
    C --> G
    C --> H
    C --> I
    C --> J
    
    H --> E
    I --> F
    J --> F
    J --> G
```

### 5. Data Flow for Audit Logging

```mermaid
sequenceDiagram
    participant User as AI Agent
    participant CLI as CLI Handler
    participant Policy as Policy Engine
    participant Logger as Audit Logger
    participant FS as File System
    
    User->>CLI: agentshield run -- command
    CLI->>Policy: Evaluate(command, paths)
    Policy->>CLI: EvalResult{decision, rules, reasons}
    
    CLI->>Logger: Create Audit Event
    Note over Logger: Timestamp, Command, Args, Cwd, Decision, TriggeredRules, Mode
    
    alt Decision = Block
        CLI->>Logger: Log(event)
        Logger->>FS: Write to audit.jsonl
        CLI->>User: ❌ BLOCKED
    else Decision = Require Approval
        CLI->>User: Show approval prompt
        User->>CLI: Approve/Deny
        CLI->>Logger: Update event with UserAction
        Logger->>FS: Write to audit.jsonl
    else Decision = Sandbox
        CLI->>Logger: Log sandbox execution
        Logger->>FS: Write to audit.jsonl
    else Decision = Allow
        CLI->>Logger: Log execution
        Logger->>FS: Write to audit.jsonl
    end
```

## Key Components Explained

### Main Entry Point (`cmd/agentshield/main.go`)
- Simple entry point that delegates to CLI package
- Handles error reporting and exit codes

### CLI Package (`internal/cli/`)
- **root.go**: Defines main CLI structure using Cobra framework
- **run.go**: Core command execution logic with policy enforcement
- **version.go**: Version information

### Core Processing Packages

#### Config (`internal/config/`)
- Loads configuration from default locations or user-specified paths
- Manages policy and log file paths
- Creates `.agentshield` directory in user home

#### Policy Engine (`internal/policy/`)
- **engine.go**: Core policy evaluation logic
- **loader.go**: Policy file loading from YAML
- **types.go**: Policy data structures
- Rule matching supports exact match, prefix match, and regex patterns
- Protected path checking with glob patterns

#### Normalizer (`internal/normalize/`)
- Extracts file paths from commands
- Normalizes relative paths to absolute paths
- Handles path expansion and resolution

#### Sandbox (`internal/sandbox/`)
- Creates isolated environments for command execution
- Captures file system changes
- Provides diff summaries for user review
- Applies approved changes to real system

#### Approval (`internal/approval/`)
- Interactive user prompts for approval decisions
- Formats approval requests with rule explanations
- Captures user actions for audit logging

#### Logger (`internal/logger/`)
- Structured audit logging in JSONL format
- Tracks all command executions and decisions
- Provides security audit trail

#### Redact (`internal/redact/`)
- Sensitive data redaction for logging
- Prevents secrets from appearing in audit logs

## Development Workflow

1. **Policy Development**: Create YAML policy files with rules and protected paths
2. **Testing**: Use sandbox mode to preview changes
3. **Audit Review**: Monitor audit logs for security events
4. **Configuration**: Customize paths and modes as needed

## Security Model

- **Defense in Depth**: Multiple layers of security checks
- **Fail Safe**: Default to blocking when uncertain
- **Audit Trail**: Complete logging of all actions
- **User Control**: Approval workflows for risky operations
- **Sandboxing**: Isolated execution for preview capabilities
