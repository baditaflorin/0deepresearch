---
title: 'An Expert Technical Report on pgai: Architecture, Implementation, and Future of In-Database AI with PostgreSQL'
date: 2025-07-22T02:44:00
draft: false
description: The modern data landscape is undergoing a profound architectural transformation, characterized by a decisive shift toward in-database artificial intelligence. Historically, machine learning and data processing were disparate disciplines; data was extracted from transactional databases, moved across networks to specialized analytical platforms, processed, and the resulting insights were then moved back. This extract-transform-load (ETL) or extract-load-transform (ELT) paradigm, while functional, introduces significant latency, increases operational complexity, and creates data security vulnerabilities. The current industry trend is a direct response to these limitations, advocating for moving analytical and AI/ML capabilities directly into the database engine itself. This convergence is motivated by a desire to reduce data movement, minimize latency for real-time applications, simplify technology stacks, and, critically, leverage the robust, battle-tested security and transactional integrity of systems like PostgreSQL.1 This is not a fleeting trend but a fundamental evolution in data architecture, promising a future where data management and intelligence are seamlessly unified.4
---
## **Introduction: The Convergence of Data and Intelligence in PostgreSQL**

The modern data landscape is undergoing a profound architectural transformation, characterized by a decisive shift toward in-database artificial intelligence. Historically, machine learning and data processing were disparate disciplines; data was extracted from transactional databases, moved across networks to specialized analytical platforms, processed, and the resulting insights were then moved back. This extract-transform-load (ETL) or extract-load-transform (ELT) paradigm, while functional, introduces significant latency, increases operational complexity, and creates data security vulnerabilities. The current industry trend is a direct response to these limitations, advocating for moving analytical and AI/ML capabilities directly into the database engine itself. This convergence is motivated by a desire to reduce data movement, minimize latency for real-time applications, simplify technology stacks, and, critically, leverage the robust, battle-tested security and transactional integrity of systems like PostgreSQL.1 This is not a fleeting trend but a fundamental evolution in data architecture, promising a future where data management and intelligence are seamlessly unified.4

Within this evolving ecosystem, Timescale's pgai emerges not merely as a collection of features, but as the embodiment of an opinionated and robust architectural philosophy for building AI applications on PostgreSQL.7 It is designed to empower a new class of developer—the "AI Engineer"—who prioritizes the practical application of AI models to build production-grade software over fundamental ML research.8 The core contribution of

pgai is a resilient, data-centric pattern for AI development that abstracts away the complexities of MLOps, allowing developers to remain within the familiar and powerful confines of the PostgreSQL environment. This report provides an exhaustive technical analysis of the pgai ecosystem, its architecture, practical implementation patterns, performance characteristics, and a theoretical exploration of its future potential in building next-generation, end-to-end intelligent data pipelines.

## **I. The pgai Ecosystem: Architecture and Design Principles**

To fully grasp the capabilities of pgai, it is essential to understand that it is not a monolithic extension but a cohesive suite of layered components. Each layer serves a distinct purpose, building upon the last to create a comprehensive AI platform within PostgreSQL. This modularity is a strategic design choice, allowing developers to adopt components incrementally based on their specific needs.

### **Deconstructing the Suite: The Three Layers of AI on Postgres**

The Timescale AI stack is comprised of three primary open-source extensions: pgvector, pgvectorscale, and pgai. Together, they transform a standard PostgreSQL instance into a high-performance vector database with sophisticated AI orchestration capabilities.10

* **pgvector:** This is the foundational layer for all vector-related workloads in the PostgreSQL ecosystem. Its primary function is to introduce a vector data type and a set of fundamental operators for calculating vector similarity. These operators include cosine distance (\<=\>), L2 distance (\<-\>), and inner product (\<\#\_\>). pgvector also provides initial indexing capabilities, such as IVFFlat and Hierarchical Navigable Small Worlds (HNSW), which are crucial for accelerating similarity searches beyond simple sequential scans.10 It serves as the  
  *lingua franca* for vector storage and basic search within Postgres.  
* **pgvectorscale:** This is the performance and scalability layer, designed to elevate PostgreSQL's vector search capabilities to be competitive with, or even superior to, specialized, proprietary vector databases.13 It enhances  
  pgvector with state-of-the-art technologies, most notably a search index inspired by Microsoft's DiskANN research, called StreamingDiskANN, which is optimized for high-performance Approximate Nearest Neighbor (ANN) search on datasets that exceed available RAM.14 It also introduces advanced quantization techniques like Statistical Binary Quantization (SBQ) to reduce the memory and storage footprint of vectors, making large-scale workloads more cost-efficient.15  
* **pgai:** This is the orchestration and ease-of-use layer. It consists of two main parts: a Python library and a PostgreSQL extension.  
  * The **pgai Vectorizer** is a Python library and associated worker process that automates the creation and synchronization of vector embeddings from data in PostgreSQL or S3.7  
  * The **pgai PostgreSQL extension** provides a suite of SQL functions that allow for direct, in-database interaction with external LLM APIs (e.g., for summarization, classification, or reranking).8

This layered architecture provides developers with flexibility. A project can start with just pgvector for basic semantic search, later add pgvectorscale when performance at scale becomes a requirement, and finally integrate pgai to simplify and automate the AI development workflow.11

| Component | Primary Function | Key Features | When to Use It |
| :---- | :---- | :---- | :---- |
| **pgvector** | Vector Storage & Basic Search | vector data type, distance operators (\<=\>, \<-\>), HNSW & IVFFlat indexes. | For any application requiring vector storage and similarity search in PostgreSQL. The foundational requirement. 10 |
| **pgvectorscale** | Performance & Scalability | StreamingDiskANN index, Statistical Binary Quantization (SBQ), improved filtered search. | When dealing with large-scale vector datasets (millions+) or when low-latency, high-throughput ANN search is critical. 14 |
| **pgai Vectorizer** | Automated Embedding Management | Declarative pipeline (load, parse, chunk, format, embed), asynchronous processing, automatic synchronization. | To eliminate the MLOps complexity of keeping vector embeddings in sync with changing source data in a robust, production-ready manner. 7 |
| **pgai SQL Functions** | In-Database AI Orchestration | Direct SQL calls to external LLM APIs (e.g., OpenAI, Cohere) for tasks like summarization, classification, and reranking. | For real-time data enrichment or implementing advanced AI logic (like hybrid search) directly within SQL queries. 8 |

### **Core Design Tenet: Decoupling for Production Resilience**

The most critical and insightful aspect of the pgai architecture is its deliberate decoupling of the application's primary data operations from the embedding generation process. This design choice is fundamental to its production readiness and represents a sophisticated understanding of the failure modes inherent in distributed systems.

A naive approach to generating embeddings, often a developer's first instinct, might involve using a PostgreSQL trigger. For instance, an AFTER INSERT trigger on a documents table could call a PL/Python function that makes an HTTP request to an external embedding API (like OpenAI's). While simple, this pattern is exceptionally fragile. It creates a tight, synchronous coupling between the core business transaction (inserting the document) and an unreliable external service. Any failure in the external API call—a network timeout, a rate limit error, a 503 Service Unavailable response—would cause the entire database transaction to fail and roll back. This means an intermittent issue with a third-party service could prevent users from performing core application functions, a completely unacceptable state for a production system.7

The pgai Vectorizer architecture elegantly solves this problem. It treats embeddings not as primary, transactionally-consistent attributes, but as *derived, eventually consistent data*, much like a materialized view or a full-text search index. The process works as follows:

1. **Fast, Isolated Writes:** The application performs a standard INSERT or UPDATE on the source table (e.g., documents). This operation is fast, atomic, and completely isolated from any external services.  
2. **Internal Work Queue:** The pgai Vectorizer, when created, sets up internal tables that act as a work queue. The change to the documents table is recorded in this queue.  
3. **Asynchronous Worker Process:** A separate, long-running Python worker process (pgai-vectorizer-worker) polls this database queue for new work.7  
4. **Resilient Processing:** The worker pulls jobs from the queue in batches, makes the calls to the external embedding API, and handles the operational complexities. It includes built-in, configurable retry logic to gracefully manage transient API failures, rate limits, and latency spikes.7  
5. **Writing Back Embeddings:** Upon successfully generating an embedding, the worker writes it back to the designated embedding storage table.

This asynchronous, queued architecture ensures that the application's write path remains fast and reliable, completely insulated from the instabilities of external network calls. A temporary outage of the embedding provider will simply cause the work queue to grow; once the service is restored, the worker will catch up. This design transforms a fragile, synchronous operation into a resilient, asynchronous one, making pgai a production-ready architectural pattern for data enrichment out-of-the-box.7

## **II. Practical Implementation: From Setup to a Production-Ready RAG Pipeline**

This section provides a practical, step-by-step guide to implementing a sophisticated AI application using the pgai ecosystem. It covers environment setup, defining a vectorizer, and building both a semantic search function and a complete Retrieval-Augmented Generation (RAG) pipeline.

### **Environment Configuration and Installation**

A robust development environment is the first step. The recommended approach utilizes Docker for ease of setup and consistency.

1. **Database Setup:** Use the official Timescale Docker image, which comes pre-packaged with PostgreSQL, TimescaleDB, pgvector, pgvectorscale, and pgai. The following docker-compose.yaml file provides a simple, reproducible setup 17:  
   YAML  
   name: pgai  
   services:  
     db:  
       image: timescale/timescaledb-ha:pg17  
       environment:  
         POSTGRES\_PASSWORD: your\_strong\_password  
         \# Set API keys as environment variables for the Postgres session  
         VOYAGE\_API\_KEY: your-voyage-api-key  
         OPENAI\_API\_KEY: your-openai-api-key  
         COHERE\_API\_KEY: your-cohere-api-key  
       ports:  
         \- "5432:5432"  
       volumes:  
         \- db\_data:/var/lib/postgresql/data

   volumes:  
     db\_data:

2. **Enable Extensions:** Once the container is running, connect to the database using psql or any standard client and enable the necessary extensions. The ai extension depends on pgvector, so using CASCADE will install both.16  
   SQL  
   CREATE EXTENSION IF NOT EXISTS ai CASCADE;

3. **API Key Management:** While API keys can be passed as environment variables to the Docker container (as shown above), a more secure and flexible method for client connections is to use session-level parameters. This avoids hardcoding keys and allows them to be managed by the application's environment.18  
   Bash  
   \# Example connection from a client machine  
   PGOPTIONS="-c ai.openai\_api\_key=$OPENAI\_API\_KEY" psql \-h localhost \-U postgres

4. **The pgai Worker:** For any self-hosted PostgreSQL instance (including this Docker setup), the pgai Vectorizer requires its companion Python worker process to be running. This process is responsible for polling the database and generating embeddings. Install it via pip and run it, pointing it to your database URL.7  
   Bash  
   \# Install the library with worker dependencies  
   pip install "pgai\[vectorizer-worker\]"

   \# Run the worker to continuously poll for jobs  
   pgai-vectorizer-worker \-d "postgres://postgres:your\_strong\_password@localhost:5432/postgres"

### **The Declarative Vectorizer in Depth (ai.create\_vectorizer)**

The heart of pgai's automation is the ai.create\_vectorizer function. It defines a complete, declarative pipeline for transforming source data into vector embeddings. This pipeline consists of several configurable stages 7:

* **Loading:** Defines the data source. This is typically a column in a table (ai.loading\_column) but can also be a column containing URIs that point to external files in locations like an S3 bucket. This allows pgai to embed not just structured text but also unstructured documents associated with your database records.7  
* **Parsing:** Specifies how to handle non-textual data sourced from URIs. pgai can parse various document types, such as PDF, HTML, or Markdown, into clean text suitable for embedding. This is particularly powerful when combined with tools like unstructured, as demonstrated in pgai's example repositories.7  
* **Chunking:** Defines the strategy for splitting large text documents into smaller, semantically meaningful chunks. Effective chunking is critical for RAG performance, as embedding an entire large document can dilute its meaning.  
* **Formatting:** Allows for pre-processing each chunk before it is sent to the embedding model. A common and effective technique is to prepend metadata, like the document's title, to each chunk. This provides the embedding model with additional context, often leading to more precise vector representations.  
* **Embedding:** Configures the LLM provider, model, and parameters (e.g., embedding dimensions) to be used for generating the vectors.

### **Code Walkthrough: Building a High-Fidelity Semantic Search Application**

This example demonstrates building a semantic search system for a blog post database using Voyage AI's models.

**Step 1: Schema and Vectorizer Definition**

First, create the source table for the blog posts and then define the vectorizer. The vectorizer will read from the contents column, use the voyage-large-2-instruct model, and store the results in a new table called blog\_embeddings.17

SQL

\-- Create the source table  
CREATE TABLE blog (  
    id SERIAL PRIMARY KEY,  
    title TEXT,  
    authors TEXT,  
    contents TEXT,  
    metadata JSONB  
);

\-- Create the vectorizer  
SELECT ai.create\_vectorizer(  
    'blog'::regclass,                                  \-- Source table  
    loading \=\> ai.loading\_column('contents'),          \-- Source column  
    embedding \=\> ai.embedding\_voyageai(  
        'voyage-large-2-instruct'                      \-- Embedding model  
    ),  
    destination \=\> ai.destination\_table(  
        target\_table \=\> 'blog\_embeddings'              \-- Destination table for embeddings and chunks  
    )  
);

**Step 2: Data Ingestion and Automated Embedding**

Now, insert some data into the blog table. With the pgai-vectorizer-worker process running in the background, pgai will automatically detect these new rows, process them through the defined pipeline, and populate the blog\_embeddings table. No application-level code is needed for this step.17

SQL

INSERT INTO blog (title, authors, contents, metadata) VALUES   
('Getting Started with PostgreSQL', 'John Doe', 'PostgreSQL is a powerful, open source object-relational database system...', '{"tags": \["database", "postgresql"\]}'),  
('The Future of Artificial Intelligence', 'Dr. Alan Turing', 'As we look towards the future, artificial intelligence continues to evolve...', '{"tags": \["AI", "technology"\]}');

You can monitor the logs of the pgai-vectorizer-worker to see it pick up and process the work.

**Step 3: Implementing the Search Function**

The final step is to create a search function in your application (e.g., in Python). This function will take a user's query, embed it using the *same* Voyage AI model, and then use pgvector's cosine distance operator (\<=\>) to find the most similar chunks in the database.16

Python

import psycopg  
import voyageai

\# Initialize Voyage AI client  
vo \= voyageai.Client()

async def semantic\_search\_blog(db\_pool, query\_text: str, limit: int \= 5):  
    """  
    Performs semantic search on the blog database.  
    """  
    \# 1\. Embed the user's query using the same model as the vectorizer  
    query\_embedding \= vo.embed(  
        texts=\[query\_text\],   
        model="voyage-large-2-instruct",  
        input\_type="query"  
    ).embeddings

    \# 2\. Query the database to find the most similar chunks  
    async with db\_pool.connection() as conn:  
        async with conn.cursor() as cur:  
            \# The 'embedding' column is in blog\_embeddings, which we join back to 'blog'  
            \# to get the original metadata like title and authors.  
            await cur.execute("""  
                SELECT   
                    b.title,   
                    b.authors,  
                    be.chunk,   
                    be.embedding \<=\> %s AS distance  
                FROM blog\_embeddings AS be  
                JOIN blog AS b ON be.source\_pk \= b.id::text  
                ORDER BY distance ASC  
                LIMIT %s;  
            """, (query\_embedding, limit))  
              
            results \= await cur.fetchall()  
            return results

\# Example usage  
\# results \= await semantic\_search\_blog(my\_db\_pool, "What is the role of AI in technology?")

### **Code Walkthrough: Constructing an End-to-End RAG System**

Building a RAG system is a natural extension of the semantic search function. The goal is to use the retrieved database content as context for a generative LLM to formulate a comprehensive answer.

Python

from openai import AsyncOpenAI

\# Assume semantic\_search\_blog function from previous example exists  
\# Assume an initialized OpenAI client  
aclient \= AsyncOpenAI()

async def answer\_question\_with\_rag(db\_pool, query\_text: str):  
    """  
    Finds relevant context from the database and uses an LLM to answer a question.  
    """  
    \# 1\. Retrieve relevant chunks from the database using semantic search  
    relevant\_chunks \= await semantic\_search\_blog(db\_pool, query\_text, limit=3)

    if not relevant\_chunks:  
        return "I could not find any relevant information in the database to answer your question."

    \# 2\. Format the retrieved chunks into a context string  
    context\_str \= "\\n\\n---\\n\\n".join(  
        f"From article '{result}' by {result}:\\n{result}" for result in relevant\_chunks  
    )

    \# 3\. Construct a prompt for the generative LLM  
    system\_prompt \= "You are a helpful assistant. Answer the user's question based \*only\* on the provided context."  
    user\_prompt \= f"""  
    Context:  
    {context\_str}

    Question: {query\_text}

    Answer:  
    """

    \# 4\. Call the LLM to generate the final answer  
    response \= await aclient.chat.completions.create(  
        model="gpt-4o-mini",  
        messages=\[  
            {"role": "system", "content": system\_prompt},  
            {"role": "user", "content": user\_prompt}  
        \]  
    )

    return response.choices.message.content

\# Example usage  
\# final\_answer \= await answer\_question\_with\_rag(my\_db\_pool, "What did Alan Turing say about AI?")

This end-to-end pattern—search, retrieve, augment, generate—is the cornerstone of modern AI applications, and pgai provides the robust database-side tooling to make the "retrieve" step efficient, scalable, and easy to manage.7

| Provider | Supported Models (Examples) | Key Use Case/Strength |
| :---- | :---- | :---- |
| **OpenAI** | text-embedding-3-large, text-embedding-ada-002, gpt-4o | General purpose, high-quality embeddings and powerful generation models. 7 |
| **Cohere** | embed-english-v3.0, rerank-english-v3.0, command-r | Strong multilingual support, classification, and industry-leading reranking models. 7 |
| **Voyage AI** | voyage-large-2-instruct, voyage-code-2 | State-of-the-art embeddings optimized for RAG and code search use cases. 7 |
| **Ollama** | Llama 3, Mistral, etc. | Support for running open-source models locally, ideal for privacy-sensitive or offline applications. 7 |
| **Hugging Face** | Various models via API | Access to a vast library of open-source models for specialized tasks. 7 |
| **Cloud Providers** | Azure OpenAI, AWS Bedrock, Vertex AI | Integration with major cloud providers' managed AI services for enterprise deployments. 7 |

## **III. Advanced In-Database AI Operations via SQL**

While the pgai Vectorizer excels at asynchronous, bulk embedding operations, the pgai PostgreSQL extension unlocks a second, powerful mode of operation: synchronous, on-demand AI tasks executed directly within the lifecycle of a SQL query. This duality allows developers to choose the right tool for the job—asynchronous processing for maintaining data state and synchronous execution for real-time, interactive enrichment and analysis.

### **Real-Time Data Enrichment and Classification**

The pgai SQL functions provide a direct interface to LLM APIs, enabling developers to perform complex data processing on entire tables with a single, expressive SQL statement. This is a paradigm shift from traditional approaches that would require fetching data into an application, iterating over it, calling an API for each row, and then writing the results back to the database.

**Example: Bulk Summarization**

Imagine a user\_content table with thousands of long text entries. Generating a concise summary for each can be done with one UPDATE query using the openai\_chat\_complete function.8

SQL

UPDATE user\_content  
SET   
    summary \= ai.openai\_chat\_complete(  
        'gpt-4o-mini',  
        jsonb\_build\_array(  
            jsonb\_build\_object('role', 'system', 'content', 'You are an expert summarizer. Summarize the following text in a single, neutral sentence.'),  
            jsonb\_build\_object('role', 'user', 'content', content)  
        )  
    )\-\>'choices'\-\>0\-\>'message'\-\>\>'content'  
WHERE   
    summary IS NULL;

In this query, jsonb\_build\_array and jsonb\_build\_object construct the JSON payload required by the OpenAI API. The \-\> and \-\>\> operators are used to navigate the JSON response from the API and extract the generated text. This single query iterates over every row where summary is null and populates it, demonstrating immense power and simplicity.

**Example: Automated Content Moderation**

Similarly, pgai can be used for in-database content moderation. The openai\_moderate function interfaces with OpenAI's Moderation API. The following query flags content as blocked based on the API's response.18

SQL

UPDATE user\_content  
SET   
    blocked \= (  
        ai.openai\_moderate('text-moderation-stable', content)\-\>'results'\-\>0\-\>\>'flagged'  
    )::boolean  
WHERE   
    blocked IS NOT TRUE;

**Automation with Triggers**

For true automation, these operations can be encapsulated in PostgreSQL triggers. For example, a trigger can automatically generate an embedding and a summary whenever a new piece of content is inserted.18

SQL

\-- 1\. Create the trigger function  
CREATE OR REPLACE FUNCTION enrich\_user\_content()  
RETURNS TRIGGER AS $$  
BEGIN  
    \-- Generate embedding  
    NEW.embedding \= ai.openai\_embed('text-embedding-ada-002', NEW.content);  
      
    \-- Generate summary  
    NEW.summary \= ai.openai\_chat\_complete(  
        'gpt-4o-mini',  
        jsonb\_build\_array(  
            jsonb\_build\_object('role', 'system', 'content', 'Summarize in one sentence.'),  
            jsonb\_build\_object('role', 'user', 'content', NEW.content)  
        )  
    )\-\>'choices'\-\>0\-\>'message'\-\>\>'content';

    RETURN NEW;  
END;  
$$ LANGUAGE plpgsql;

\-- 2\. Create the trigger  
CREATE TRIGGER user\_content\_enrichment\_trigger  
BEFORE INSERT ON user\_content  
FOR EACH ROW  
EXECUTE FUNCTION enrich\_user\_content();

**Important Caveat:** While powerful, using synchronous AI calls within triggers introduces latency to INSERT operations. This approach is suitable for workflows where a slight delay is acceptable. For high-throughput systems requiring sub-second insert latency, it is better to stick with the asynchronous pgai Vectorizer for embeddings and run summarization or other enrichment tasks as periodic background jobs.18

### **Implementing Hybrid Search**

For many applications, relying solely on semantic search is insufficient. A query might contain specific keywords, product codes, or names that must be matched exactly. Hybrid search, which combines traditional keyword-based search with modern semantic search, often yields the most relevant results. pgai facilitates the implementation of sophisticated hybrid search pipelines directly in SQL, especially when combined with a reranking model.

The following example demonstrates a three-step hybrid search process using PostgreSQL's Full-Text Search (FTS), pgvector's semantic search, and Cohere's Rerank API via pgai.16

SQL

WITH user\_query AS (  
    SELECT   
        'What are the best AI-powered database technologies?' AS text,  
        \-- Embed the query for the semantic search step  
        ai.cohere\_embed('embed-english-v3.0', 'What are the best AI-powered database technologies?', input\_type \=\> 'search\_query') AS embedding  
),  
\-- Step 1: Combine Keyword and Semantic Search for initial candidate retrieval  
candidate\_search AS (  
    SELECT   
        id,  
        title,  
        article,  
        \-- Use a combination of FTS rank and semantic distance for initial ordering  
        (ts\_rank(tsv, to\_tsquery('english', 'ai & database & technology'))) \+ (1 \- (embedding \<=\> (SELECT embedding FROM user\_query))) AS initial\_score  
    FROM   
        cnn\_daily\_mail \-- Assuming this table has embeddings and a 'tsv' column for FTS  
    WHERE   
        tsv @@ to\_tsquery('english', 'ai & database & technology')  
    ORDER BY   
        initial\_score DESC  
    LIMIT 100 \-- Retrieve a larger set of candidates for reranking  
)  
\-- Step 2: Use a powerful reranking model on the top candidates for final ordering  
SELECT   
    reranked.id,  
    reranked.title,  
    reranked.relevance\_score  
FROM   
    ai.cohere\_rerank(  
        'rerank-english-v3.0',  
        (SELECT text FROM user\_query),  
        (SELECT array\_agg(article) FROM candidate\_search),  
        top\_n \=\> 10 \-- Return the final top 10 results  
    ) AS reranked  
JOIN   
    candidate\_search ON reranked.input\_id \= candidate\_search.id  
ORDER BY   
    reranked.relevance\_score DESC;

This query demonstrates a powerful, multi-stage retrieval strategy:

1. A Common Table Expression (CTE) user\_query prepares the query text and its embedding.  
2. The candidate\_search CTE performs a combined search, filtering with FTS (@@ operator) and scoring based on a mix of FTS rank and semantic similarity. It retrieves the top 100 candidates.  
3. The final SELECT statement passes these 100 candidates to the ai.cohere\_rerank function. This function makes a single API call to Cohere's powerful reranker model, which re-evaluates the relevance of each candidate against the original query and returns a new, more accurate relevance score. The final results are the top 10 most relevant documents as determined by the reranker.

## **IV. Performance, Scalability, and Operational Dynamics**

While ease of use is a major feature of pgai, its viability for production systems hinges on performance and scalability. This is where pgvectorscale, the performance layer of the Timescale AI stack, becomes critical. It introduces innovations that allow PostgreSQL to handle vector workloads at a scale previously thought to require specialized databases.

### **Dissecting pgvectorscale**

pgvectorscale enhances pgvector with two key technologies designed to tackle the challenges of large-scale ANN search and storage.15

1. **StreamingDiskANN Index:** Traditional in-memory ANN indexes like HNSW perform exceptionally well but require the entire index and vector dataset to fit in RAM. As datasets grow into the tens or hundreds of millions of vectors, this becomes prohibitively expensive or impossible. StreamingDiskANN, inspired by Microsoft research, is a disk-based graph index. It intelligently caches parts of the index graph in memory while keeping the bulk of the vectors on cost-effective SSD storage. This allows it to provide high-performance, low-latency ANN search on datasets much larger than available RAM, effectively breaking the memory barrier for vector search in PostgreSQL.10  
2. **Statistical Binary Quantization (SBQ):** Vector embeddings, especially from modern models, can be large (e.g., 1536 dimensions of 32-bit floats consume 6 KB per vector). SBQ is a product quantization technique developed by Timescale that compresses these vectors into a much smaller binary representation. This significantly reduces the storage and memory footprint of the dataset, leading to lower costs and faster query times due to reduced I/O. Unlike standard binary quantization, SBQ uses statistical properties of the dataset to create more accurate compressed representations, thereby maintaining high recall (search accuracy) even after compression.15

### **Evaluation of Performance Benchmarks**

Timescale has published benchmarks comparing a PostgreSQL instance equipped with pgvector and pgvectorscale against Pinecone, a leading specialized vector database. On a dataset of 50 million 768-dimension Cohere embeddings, the results were compelling 15:

* Compared to Pinecone's storage-optimized s1 index, PostgreSQL achieved **28x lower p95 latency** and **16x higher query throughput** at 99% recall.  
* Even when compared to Pinecone's performance-optimized p2 index, PostgreSQL showed **4x lower p95 latency** and **1.5x higher query throughput** at 90% recall.  
* In terms of cost, self-hosting the PostgreSQL setup was found to be **4-5x cheaper** than using the equivalent Pinecone service.21

These benchmarks suggest that with pgvectorscale, performance and scale are no longer valid reasons to choose a separate, specialized vector database over the operational simplicity and rich feature set of PostgreSQL.13

### **Operationalizing the pgai Worker**

For production deployments, the pgai vectorizer worker process must be managed as a long-running, reliable service.

* **Deployment Strategies:** The worker can be deployed in various ways depending on the infrastructure. Common patterns include running it as a systemd service on a VM, deploying it as a dedicated pod in a Kubernetes cluster, or managing it with a process supervisor like supervisord.  
* **Scalability:** The pgai worker architecture is designed for horizontal scalability. You can run multiple worker processes, even on different machines. They will coordinate their work through the database's job queue, effectively parallelizing the embedding workload.  
* **Monitoring:** To ensure the health of the embedding pipeline, it is crucial to monitor key metrics from the worker and the database queue. These include the queue depth (number of unprocessed items), the processing rate (embeddings generated per minute), and the API error rate. These metrics can be exposed via a monitoring agent or by directly querying the pgai internal tables.

A comment was noted that the requirement of a separate Python worker is a "huge regression".23 This perspective, however, overlooks the fundamental architectural benefit this design provides. As detailed in Section I, this separation is a deliberate and sophisticated choice that ensures production resilience. By moving the unreliable, slow, and I/O-bound task of calling external APIs out of the database transaction path and into a separate, scalable, and resilient worker process,

pgai provides a far more robust architecture than a tightly-coupled in-database solution would. It is not a regression, but rather a mature design pattern for building fault-tolerant distributed systems.

## **V. A Comparative Analysis of In-Database AI Approaches**

Choosing the right architecture for an AI application on PostgreSQL involves understanding the trade-offs between different approaches. The primary alternatives are a manual implementation using pgvector and application code, pgai, and PostgresML.

### **pgai vs. The Manual Approach (pgvector \+ Application Code)**

A "manual" approach involves using pgvector for storage and search, but building all the surrounding logic in an external application. This includes code for chunking documents, calling embedding APIs, managing a queue for updates, handling API errors and retries, and keeping embeddings synchronized with source data.

* **Manual Approach:** This offers maximum flexibility. The developer has complete control over every aspect of the AI pipeline, can use any programming language, framework, or AI model, and can implement highly custom logic. However, this flexibility comes at the cost of significant, non-trivial engineering effort. Building a resilient, scalable, and maintainable pipeline to handle these tasks is a complex MLOps challenge in itself.13  
* **pgai Approach:** pgai radically simplifies this process by providing an opinionated but robust solution that handles all the "plumbing." The pgai Vectorizer abstracts away the complexity of chunking, queuing, error handling, and synchronization. This allows developers to focus on their core application logic instead of reinventing a complex data pipeline. The trade-off is a potential reduction in flexibility for highly bespoke workflows or for models not yet supported by pgai's declarative syntax.7

### **pgai vs. PostgresML**

pgai and PostgresML are two powerful but philosophically different extensions for AI in PostgreSQL. The choice between them is not about which is "better," but about which architectural philosophy aligns with the project's goals.

* **Core Philosophy:** pgai is primarily focused on simplifying the *retrieval* and *API interaction* aspects of AI. It is an ideal tool for developers building RAG and agentic applications that leverage powerful, often proprietary, external models via API calls.8 In contrast,  
  PostgresML aims to be a more comprehensive, in-database MLOps platform. It supports in-database model *training* (using algorithms from Scikit-learn, XGBoost, etc.) and inference, with a strong focus on bringing open-source models from hubs like Hugging Face *inside* the database to run on GPUs co-located with the data.24  
* **External vs. Internal Models:** This is the clearest distinction. pgai is built around seamless integration with external, best-in-class API providers like OpenAI, Cohere, and Anthropic.7  
  PostgresML, on the other hand, explicitly states that it does *not* currently support direct integration with remote LLM providers like OpenAI. Its philosophy is centered on self-hosting models within the database environment to avoid data movement and external dependencies.24  
* **Target User:** pgai is designed for the "AI Engineer" or application developer who wants to leverage powerful external AI models without leaving the familiar Postgres ecosystem.8  
  PostgresML appears to cater more to a data scientist or ML engineer who wants to perform the entire ML lifecycle, from training to deployment, entirely within the security and control of their database.

The decision between pgai and PostgresML reflects a broader strategic choice in enterprise AI: embracing the API economy of best-of-breed external models versus championing data and model sovereignty through self-hosting. For building a RAG application with GPT-4o, pgai is the purpose-built tool. For training a custom fraud detection model on internal tabular data with XGBoost, PostgresML is the more appropriate choice. They are complementary, not just competitive, solutions within the growing PostgreSQL AI ecosystem.

| Criterion | pgvector \+ App Code | pgai | PostgresML |
| :---- | :---- | :---- | :---- |
| **Primary Use Case** | Custom AI pipelines requiring maximum flexibility. | Building RAG, semantic search, and agentic apps using external LLM APIs. | In-database model training, fine-tuning, and inference with self-hosted models. |
| **Ease of Use** | Low. Requires extensive custom development for MLOps. | High. Declarative, automated embedding management and simple SQL functions. | Medium. SQL-based, but requires understanding of ML concepts and algorithms. |
| **Resilience** | Dependent on custom implementation. Often fragile if not expertly built. | High. Built-in queuing, retries, and decoupling for production readiness. | High. Keeps all operations within the database, avoiding external network failures. |
| **Model Support** | Any model accessible via an API or library. | Focused on external API providers (OpenAI, Cohere, Ollama, etc.). | Focused on internal, self-hosted models (Hugging Face, Scikit-learn, etc.). |
| **Performance** | Dependent on implementation. | High, especially when combined with pgvectorscale. | High for inference, as models are co-located with data. Can leverage GPUs. |
| **Target Persona** | Senior ML Engineer, Systems Architect. | Application Developer, "AI Engineer". | Data Scientist, ML Engineer. |
| **Key Trade-off** | Flexibility vs. Complexity. | Simplicity vs. Opinionated Workflow. | Sovereignty vs. Access to Proprietary Models. |

## **VI. Theorizing the Future: Advanced Use Cases and End-to-End Pipelines**

The true potential of pgai extends beyond standard RAG implementations. Its components can serve as building blocks for highly sophisticated, end-to-end intelligent data pipelines and autonomous systems operating entirely within the database.

### **Use Case 1: High-Throughput, Intelligent Data Processing (An Augmented dblink Pipeline)**

The provided code snippet for fill\_missing\_countries\_final\_parallel demonstrates a sophisticated, if complex, use of dblink to manually parallelize a data enrichment task. This pattern, while effective for its purpose, can be augmented with pgai to create a far more intelligent processing pipeline without a full architectural rewrite.

Consider a scenario where a stream of tedx\_speakers data requires not just country filling, but also real-time topic classification of the talk abstract and a generated one-sentence summary. A future, augmented pipeline could look like this:

1. **Ingestion:** Raw speaker data is inserted into a staging\_speakers table.  
2. **Parallel Pre-processing:** The existing dblink-based procedure is triggered. Its workers run in parallel to perform initial, non-AI tasks: data validation, cleaning, and perhaps the original country-filling logic. This leverages the existing investment in the parallel processing code.  
3. **Promotion to Enriched Table:** Upon successful pre-processing, the data is moved from the staging table to a final enriched\_speakers table.  
4. **Synchronous AI Enrichment:** The enriched\_speakers table has a BEFORE INSERT trigger. This trigger calls a function that uses pgai's synchronous SQL functions. With two simple calls to ai.openai\_chat\_complete, it can classify the talk abstract into one of several predefined categories and generate a concise summary, populating new topic and summary columns directly within the insertion transaction.18

This hybrid approach demonstrates how pgai can act as an intelligent "last-mile" enrichment layer, augmenting existing complex workflows and adding significant value without requiring external microservices or a complete architectural overhaul.

### **Use Case 2: Architecting a Natural Language Query (NLQ) Interface**

pgai provides all the necessary components to build a powerful "Text-to-SQL" system, or Natural Language Query (NLQ) interface, directly on top of PostgreSQL. This would allow non-technical users to query complex databases by asking questions in plain English.28

The end-to-end pipeline for such a system would be a sequence of in-database operations:

| Pipeline Stage | Description | Key pgai/Postgres Function | Example |
| :---- | :---- | :---- | :---- |
| **1\. User Input** | User submits a query in natural language via an application interface. | N/A | "Show me the top 5 speakers from France last year" |
| **2\. Schema Retrieval** | The system retrieves metadata (table/column names, comments) for relevant database objects. This metadata has been pre-embedded using the pgai Vectorizer. | SELECT... FROM pg\_catalog... | tables: speakers(id, name, country, date) |
| **3\. Contextual Embedding** | A semantic search is performed against the embedded schema metadata to find the tables and columns most relevant to the user's query. | embedding \<=\> ai.openai\_embed(...) | Finds that speakers.country and speakers.date are relevant. |
| **4\. Prompt Engineering** | A detailed prompt is constructed for a powerful LLM, including the user's question, the retrieved schema context, and instructions to generate valid SQL. | String concatenation in PL/pgSQL | Prompt: "Given schema \[..\], translate 'Show me top 5 speakers from France last year' to SQL." |
| **5\. SQL Generation** | The prompt is sent to an LLM via a synchronous pgai call. The LLM returns a SQL query string. | ai.openai\_chat\_complete(...) | Returns SELECT name FROM speakers WHERE country \= 'France' AND date \>= '...' LIMIT 5; |
| **6\. Query Validation & Execution** | The generated SQL is optionally validated (e.g., with EXPLAIN) and then executed against the database. | EXECUTE generated\_sql; | The query is run, returning a list of speaker names. |
| **7\. Result Summarization** | The raw tabular result is fed back into the LLM with a prompt to summarize it in a human-readable sentence. | ai.openai\_chat\_complete(...) | Prompt: "Summarize this result: \[name1, name2,...\]." |
| **8\. Final Response** | The summarized, natural language answer is returned to the user. | N/A | "The top speakers from France last year were: \[name1\], \[name2\],..." |

This entire, sophisticated workflow can be orchestrated within PostgreSQL stored procedures, leveraging pgai for all AI interactions, creating a self-contained and powerful NLQ engine.

### **Use Case 3: Autonomous Database Agents**

Pushing the theoretical boundaries further, pgai enables the creation of simple, autonomous agents that operate within the database to perform maintenance, optimization, and quality control tasks, reflecting the future of AI-driven database management.4

* **Schema Documentation Agent:** A scheduled job (using pg\_cron or TimescaleDB's user-defined actions) could iterate through all tables in a schema. For each table, it would feed the CREATE TABLE DDL and a sample of rows to an LLM via pgai with the prompt, "Generate a concise, human-readable description for each column in this table based on its name and data." The agent would then write the LLM's output directly into the database's metadata store using COMMENT ON COLUMN... IS '...'.  
* **Query Optimization Agent:** An agent could periodically analyze the pg\_stat\_statements view to identify the most frequent or slowest queries. It would then pass the query text and the relevant EXPLAIN plan to an LLM with a prompt like, "You are an expert PostgreSQL DBA. Analyze this query and its execution plan. Suggest an alternative query structure or a missing index to improve its performance." The suggestions could be logged for review by a human DBA.  
* **Data Quality Agent:** An agent could sample rows from a product catalog and use an LLM for semantic validation. For example, it could pass a product's title and description to an LLM and ask, "Does this description accurately and relevantly describe the product named in the title? Answer with only 'YES' or 'NO'." Rows flagged as 'NO' could be marked for manual review, catching subtle data quality issues that traditional constraints would miss.

## **Conclusion: Strategic Recommendations and Final Assessment**

This comprehensive analysis reveals that pgai and its companion extensions, pgvector and pgvectorscale, represent a mature and powerful ecosystem for building sophisticated AI applications directly within PostgreSQL.

Summary of Findings:  
The core strength of pgai lies in its production-ready architecture, which is founded on the principle of decoupling the application's write path from the asynchronous embedding pipeline. This design ensures resilience and performance, addressing a critical failure mode of naive in-database AI implementations. The pgai suite successfully simplifies the "AI Engineer's" workflow, abstracting away complex MLOps tasks and allowing developers to leverage powerful external LLMs through familiar SQL and declarative configurations. Furthermore, performance benchmarks demonstrate that when augmented with pgvectorscale, PostgreSQL can achieve throughput and latency on par with, or even superior to, specialized vector databases, all while offering significant cost advantages and the operational simplicity of a unified data stack.15  
Ideal Application Domains:  
Based on this analysis, pgai is the optimal architectural choice for a range of modern AI applications:

* **Retrieval-Augmented Generation (RAG) and Semantic Search:** pgai is purpose-built for this domain, providing an end-to-end, automated solution for creating, synchronizing, and searching vector embeddings at scale.  
* **AI-Powered Data Enrichment Pipelines:** For applications that need to enrich data with summaries, classifications, tags, or content moderation flags, pgai's SQL functions offer a remarkably simple and powerful method for performing these tasks in bulk or in real-time.  
* **Rapid Prototyping and Production Deployment:** For teams that want to build applications leveraging leading LLM APIs (from OpenAI, Cohere, Anthropic, etc.) without the overhead of managing external microservices for AI orchestration, pgai provides a direct and efficient path from prototype to production within a Postgres-centric stack.

Final Assessment:  
The pgai ecosystem is a significant and well-architected step in the evolution of intelligent database systems. It successfully and elegantly bridges the gap between the robust, transactional world of PostgreSQL and the dynamic, fast-moving world of generative AI. By providing a resilient, performant, and developer-friendly set of tools, pgai makes PostgreSQL not just a viable option, but arguably a superior choice for a wide and growing range of modern AI applications. It empowers developers to build the next generation of intelligent software on a foundation they already know and trust.

#### **Works cited**

1. In-Database Machine Learning with HeatWave AutoML \- Oracle, accessed July 22, 2025, [https://www.oracle.com/database/in-database-machine-learning/](https://www.oracle.com/database/in-database-machine-learning/)  
2. 4\. In-Database Machine Learning \- Accelerate Machine Learning with a Unified Analytics Architecture \[Book\] \- O'Reilly Media, accessed July 22, 2025, [https://www.oreilly.com/library/view/accelerate-machine-learning/9781098120313/ch04.html](https://www.oreilly.com/library/view/accelerate-machine-learning/9781098120313/ch04.html)  
3. Comparing Traditional and In-Database Machine Learning \- Ocient, accessed July 22, 2025, [https://ocient.com/blog/in-database-ml-versus-traditional-ml-which-is-right-for-your-business/](https://ocient.com/blog/in-database-ml-versus-traditional-ml-which-is-right-for-your-business/)  
4. The AI Revolution In Infrastructure And Database Management \- Forbes, accessed July 22, 2025, [https://www.forbes.com/councils/forbestechcouncil/2025/07/17/the-ai-revolution-in-infrastructure-and-database-management/](https://www.forbes.com/councils/forbestechcouncil/2025/07/17/the-ai-revolution-in-infrastructure-and-database-management/)  
5. How Generative AI Is Changing the Way We Work With Databases | Built In, accessed July 22, 2025, [https://builtin.com/articles/generative-ai-database-management](https://builtin.com/articles/generative-ai-database-management)  
6. Database Trends and Innovations: A Comprehensive Outlook for 2025 \- Rapydo, accessed July 22, 2025, [https://www.rapydo.io/blog/database-trends-and-innovations-a-comprehensive-outlook-for-2025](https://www.rapydo.io/blog/database-trends-and-innovations-a-comprehensive-outlook-for-2025)  
7. timescale/pgai: A suite of tools to develop RAG, semantic search, and other AI applications more easily with PostgreSQL \- GitHub, accessed July 22, 2025, [https://github.com/timescale/pgai](https://github.com/timescale/pgai)  
8. Pgai: Giving PostgreSQL Developers AI Engineering Superpowers \- TigerData, accessed July 22, 2025, [https://www.tigerdata.com/blog/pgai-giving-postgresql-developers-ai-engineering-superpowers](https://www.tigerdata.com/blog/pgai-giving-postgresql-developers-ai-engineering-superpowers)  
9. Pgai Brings Your ML Workload To The Database \- I Programmer, accessed July 22, 2025, [https://www.i-programmer.info/news/80-java/17332-pgai-brings-your-ml-workload-to-the-database.html](https://www.i-programmer.info/news/80-java/17332-pgai-brings-your-ml-workload-to-the-database.html)  
10. Power your AI apps with pgai on Tiger Cloud \- Timescale documentation, accessed July 22, 2025, [https://docs.timescale.com/ai/latest/?\_\_hstc=231067136.6694e1c5b8259356fcccdd9cfcb617fb.1750377600244.1750377600245.1750377600246.1&\_\_hssc=231067136.6.1750377600247&\_\_hsfp=150561067](https://docs.timescale.com/ai/latest/?__hstc=231067136.6694e1c5b8259356fcccdd9cfcb617fb.1750377600244.1750377600245.1750377600246.1&__hssc=231067136.6.1750377600247&__hsfp=150561067)  
11. Power your AI apps with pgai on Tiger Cloud \- Docs \- TigerData, accessed July 22, 2025, [https://docs.tigerdata.com/ai/latest/](https://docs.tigerdata.com/ai/latest/)  
12. PostgreSQL for AI applications \- Ubuntu, accessed July 22, 2025, [https://ubuntu.com/blog/postgresql-ai-application](https://ubuntu.com/blog/postgresql-ai-application)  
13. Timescale is making PostgreSQL better for AI ⚡️ \- Cerebral Valley, accessed July 22, 2025, [https://cerebralvalley.ai/blog/timescale-is-making-postgresql-better-for-ai-1tiUqSzGsSn76ORZVfMwOk](https://cerebralvalley.ai/blog/timescale-is-making-postgresql-better-for-ai-1tiUqSzGsSn76ORZVfMwOk)  
14. Pgai on Timescale x LlamaIndex: Making PostgreSQL a Better Vector Database for AI Applications \- TigerData, accessed July 22, 2025, [https://www.tigerdata.com/blog/timescale-vector-x-llamaindex-making-postgresql-a-better-vector-database-for-ai-applications](https://www.tigerdata.com/blog/timescale-vector-x-llamaindex-making-postgresql-a-better-vector-database-for-ai-applications)  
15. Making PostgreSQL a Better AI Database \- TigerData, accessed July 22, 2025, [https://www.tigerdata.com/blog/making-postgresql-a-better-ai-database](https://www.tigerdata.com/blog/making-postgresql-a-better-ai-database)  
16. Build Search and RAG Systems on PostgreSQL Using Cohere\&Pgai ..., accessed July 22, 2025, [https://www.tigerdata.com/blog/build-search-and-rag-systems-on-postgresql-using-cohere-and-pgai](https://www.tigerdata.com/blog/build-search-and-rag-systems-on-postgresql-using-cohere-and-pgai)  
17. pgai/docs/vectorizer/quick-start-voyage.md at main · timescale/pgai ..., accessed July 22, 2025, [https://github.com/timescale/pgai/blob/main/docs/vectorizer/quick-start-voyage.md](https://github.com/timescale/pgai/blob/main/docs/vectorizer/quick-start-voyage.md)  
18. Using AI directly from your database \- with PostgreSQL and pgai, accessed July 22, 2025, [https://www.pondhouse-data.com/blog/ai-directly-from-your-database](https://www.pondhouse-data.com/blog/ai-directly-from-your-database)  
19. timescale/unstructured-pgai-example \- GitHub, accessed July 22, 2025, [https://github.com/timescale/unstructured-pgai-example](https://github.com/timescale/unstructured-pgai-example)  
20. 18 Months of Pgvector Learnings in 47 Minutes (Tutorial) \- YouTube, accessed July 22, 2025, [https://www.youtube.com/watch?v=Ua6LDIOVN1s](https://www.youtube.com/watch?v=Ua6LDIOVN1s)  
21. Timescale Debuts Two Open Source Extensions for Making PostgreSQL Better for AI Use Cases \- Database Trends and Applications, accessed July 22, 2025, [https://www.dbta.com/Editorial/News-Flashes/Timescale-Debuts-Two-Open-Source-Extensions-for-Making-PostgreSQL-Better-for-AI-Use-Cases-164471.aspx](https://www.dbta.com/Editorial/News-Flashes/Timescale-Debuts-Two-Open-Source-Extensions-for-Making-PostgreSQL-Better-for-AI-Use-Cases-164471.aspx)  
22. How much is too much to consider pgvector : r/vectordatabase \- Reddit, accessed July 22, 2025, [https://www.reddit.com/r/vectordatabase/comments/1b1ixkq/how\_much\_is\_too\_much\_to\_consider\_pgvector/](https://www.reddit.com/r/vectordatabase/comments/1b1ixkq/how_much_is_too_much_to_consider_pgvector/)  
23. Stop over-engineering AI apps: just use Postgres : r/LocalLLaMA \- Reddit, accessed July 22, 2025, [https://www.reddit.com/r/LocalLLaMA/comments/1isiyl1/stop\_overengineering\_ai\_apps\_just\_use\_postgres/](https://www.reddit.com/r/LocalLLaMA/comments/1isiyl1/stop_overengineering_ai_apps_just_use_postgres/)  
24. postgresml/postgresml: Postgres with GPUs for ML/AI apps. \- GitHub, accessed July 22, 2025, [https://github.com/postgresml/postgresml](https://github.com/postgresml/postgresml)  
25. PostgresML Tutorial: Doing Machine Learning With SQL \- DataCamp, accessed July 22, 2025, [https://www.datacamp.com/tutorial/postgresml-tutorial-machine-learning-with-sql](https://www.datacamp.com/tutorial/postgresml-tutorial-machine-learning-with-sql)  
26. pgai·PyPI, accessed July 22, 2025, [https://pypi.org/project/pgai/0.9.1/](https://pypi.org/project/pgai/0.9.1/)  
27. What is Real-Time ML and Why Does Stream Processing Matter \- Bytewax, accessed July 22, 2025, [https://bytewax.io/blog/real-time-ml](https://bytewax.io/blog/real-time-ml)  
28. What is a Natural Language Query (NLQ)? \- AtScale, accessed July 22, 2025, [https://www.atscale.com/glossary/natural-language-query-nlq/](https://www.atscale.com/glossary/natural-language-query-nlq/)  
29. What Is Natural Language Querying? \- Ontotext, accessed July 22, 2025, [https://www.ontotext.com/knowledgehub/fundamentals/what-is-natural-language-querying/](https://www.ontotext.com/knowledgehub/fundamentals/what-is-natural-language-querying/)  
30. Natural Language Interface for Data Platform | ABCloudz, accessed July 22, 2025, [https://abcloudz.com/blog/natural-language-interface-for-data-platforms/](https://abcloudz.com/blog/natural-language-interface-for-data-platforms/)  
31. Future of Database Analyzing with AI \- DBInsights, accessed July 22, 2025, [https://dbinsights.ai/future-of-database-analyzing-with-ai/](https://dbinsights.ai/future-of-database-analyzing-with-ai/)
