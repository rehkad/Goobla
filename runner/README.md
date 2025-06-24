# `runner`

A minimal HTTP server for loading a model and running inference.

Run the server with:
```shell
./runner -model <model binary>
```

### Completion

Send a completion request. Responses are streamed as JSON objects.
```shell
curl -X POST -H "Content-Type: application/json" -d '{"prompt": "hi"}' http://localhost:8080/completion
```

### Embeddings

```shell
curl -X POST -H "Content-Type: application/json" -d '{"prompt": "turn me into an embedding"}' http://localhost:8080/embedding
```

### Health

```shell
curl http://localhost:8080/health
```
