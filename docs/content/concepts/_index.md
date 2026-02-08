---
title: "Concepts"
weight: 2
---

# Core Concepts

Understanding the key concepts behind embapi helps you make the most of its features. This section explains the fundamental building blocks and how they work together.

## Overview

embapi is a vector database designed for Retrieval Augmented Generation (RAG) workflows. It stores embeddings with metadata and provides fast similarity search capabilities.

## Key Components

- **[Users](users-and-auth/)** - Individual accounts with authentication
- **[Projects](projects/)** - Containers for embeddings with access control
- **[Embeddings](embeddings/)** - Vector representations of text with metadata
- **[LLM Services](llm-services/)** - Configurations for embedding models
- **[Similarity Search](similarity-search/)** - Find similar documents using vector distance
- **[Metadata](metadata/)** - Structured data with validation and filtering
- **[Architecture](architecture/)** - Technical architecture and design

## Architecture

embapi uses PostgreSQL with the pgvector extension for vector operations. It provides a RESTful API with token-based authentication and supports multi-user environments with project sharing.
