### Example

```sh
./mq-cli -mqhost=l:1234 --publish ping:ping
```

With data:

```sh
./mq-cli -mqhost=l:1234 --publish ping:ping --data '{"a": "b"}'
```

#### Arena-master > launch a game

```sh
./mq-cli -mqhost=l:1234 --publish arena:launch --data '{"id": "5"}'
```

