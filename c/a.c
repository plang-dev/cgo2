
int write(int fd, void *buf, int size);

char abc[5] = {"123"};

void fa() {
    write(1, "hello", 5);
}

void fb() {
    write(1, "world", 5);
}
