### Generate proto file with Buf

[Install buf](https://buf.build/docs/installation)

####   Use buf cli
```
cd proto
buf lint
buf generate
```

#### Use go generate
```
go generate ./...
```