class CustomNewlineReader:
    def __init__(self, fileobj, newline):
        self.fileobj = fileobj
        if len(newline) != 1:
            raise ValueError("newline must be exactly one character long")
        self.newline = newline
        self.buffer_size = 1 << 18
        self._chunks = []

    def __iter__(self):
        return self

    def __next__(self):
        if self._chunks:
            return self._chunks.pop(0)
        read = self.fileobj.read
        buffer = read(self.buffer_size)
        if not buffer:
            raise StopIteration
        extra = bytearray()
        b = buffer[-1:]
        while b != self.newline:
            b = read(1)
            if not b:
                break
            extra.append(b[0])
        buffer += extra
        self._chunks = buffer.split(self.newline)
        if b == self.newline:
            self._chunks = self._chunks[:-1]
        return next(self)
