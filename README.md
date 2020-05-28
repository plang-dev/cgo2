# cgo2

Usage:

```
$ go get github.com/plang-dev/cgo2
$ cat a.c
int write(int fd, void *buf, int size);

void fa() {
    write(1, "hello", 5);
}

$ clang -o a.o -c a.c
$ cgo2 a.o _fa
hello
```

