---
title: "Concepts"
weight: 2
---

# Core Concepts

Understanding the key concepts behind embapi helps you make the most of its features. This section explains the fundamental building blocks and how they work together.

## Overview

embapi is a vector database designed for Retrieval Augmented Generation (RAG) workflows. It stores embeddings with metadata and provides fast similarity search capabilities.

## Key Components

- **Users** - Individual accounts with authentication
- **Projects** - Containers for embeddings with access control
- **Embeddings** - Vector representations of text with metadata
- **LLM Services** - Configurations for embedding models
- **Similarity Search** - Find similar documents using vector distance

## Architecture

embapi uses PostgreSQL with the pgvector extension for vector operations. It provides a RESTful API with token-based authentication and supports multi-user environments with project sharing.
