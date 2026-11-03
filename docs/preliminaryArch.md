```mermaid
---
title: New Architecture
---

flowchart TB
    Parser
    Caching
    Graph("Graph Model")
    Rendering
    CLI
    Input
    Config
    
    CLI-->Config
    CLI-->Input
    CLI-->Parser
    CLI-->Rendering
    Parser-->Caching
    Parser-->Graph
    Parser-->Config
    Rendering-->Graph
```
