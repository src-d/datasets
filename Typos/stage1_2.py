import lzma

from tqdm import tqdm


with open("result.txt", "w") as fout:
    with lzma.open("repos.txt.xz", "rb") as repos:
        with open("commits.bin", "rb") as commits:
            with open("candidates.txt") as indices:
                indices.seek(0, 2)
                with tqdm(total=indices.tell()) as progress:
                    indices.seek(0, 0)
                    extra = b""
                    ri = 0
                    parts = []
                    line = indices.readline()
                    index = 0
                    commit = b""

                    def scan():
                        global parts, ri, index, commit
                        if ri + len(parts) > index:
                            parts = parts[index - ri:]
                            ri = index
                            fout.write("%s %s\n" % (commit, parts[0].decode()))
                            return True
                        ri += len(parts)
                        parts = []
                        return False

                    while line:
                        progress.n = indices.tell()
                        progress.update(0)
                        new_index = int(line)
                        commit = commits.read(20 * (new_index - index + int(not index)))[-20:].hex()
                        index = new_index

                        while not scan():
                            chunk = repos.read(1 << 18)
                            if len(chunk) != 1 << 18:
                                break
                            parts = chunk.split(b"\0")
                            parts[0] = extra + parts[0]
                            extra = parts[-1]
                            parts = parts[:-1]
                        line = indices.readline()
