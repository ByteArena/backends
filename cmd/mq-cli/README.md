### Example

```sh
./mq-cli -mqhost=l:1234 --publish ping:ping
```

With data:

```sh
./mq-cli -mqhost=l:1234 --publish ping:ping --data '{"a": "b"}'
```
