package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"unicode"
)

func main() {
	archive := os.Args[1]
	ar := os.Args[2]
	nm := os.Args[3]
	ranlib := os.Args[4]

	if err := exec.Command(ar, "x", archive, "go.o").Run(); err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	c := exec.Command(nm, "go.o")
	c.Stdout = buf
	if err := c.Run(); err != nil {
		panic(err)
	}

	if err := exec.Command(ar, "rs", archive, "go.o").Run(); err != nil {
		panic(err)
	}

	t := newTrie()

	content := bytes.Split(buf.Bytes(), []byte{'\n'})
	for _, line := range content {
		splitted := bytes.Split(line, []byte{' '})
		if len(splitted) >= 3 {
			symbol := splitted[2]
			if bytes.Index(bytes.ToLower(symbol), []byte("frida")) == -1 {
				if bytes.Index(symbol, []byte("type:.eq.")) != -1 {
					closeIdx := bytes.Index(symbol, []byte("]"))
					if bytes.Index(symbol, []byte("[")) != -1 &&
						closeIdx != -1 {
						slicedSymbol := symbol[closeIdx+1:]
						t.insert(slicedSymbol, flipAlpha(slicedSymbol))
					} else {
						slicedSymbol := symbol[9:] // count of type:.eq. is 9
						t.insert(slicedSymbol, flipAlpha(slicedSymbol))
					}
				} else {
					t.insert(symbol, flipAlpha(symbol))
				}
			}
		}
	}

	f, err := os.OpenFile(archive, os.O_RDWR, 0755)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	data, _ := io.ReadAll(f)
	modifiedData := make([]byte, len(data))

	for i := 0; i < len(data); {
		if l, replacement, ok := t.search(data, i); ok {
			copy(modifiedData[i:], replacement)
			i += l
		} else {
			modifiedData[i] = data[i]
			i++
		}
	}

	f.Truncate(0)
	f.Seek(0, 0)
	f.Write(modifiedData)

	ranl := exec.Command(ranlib, archive)
    if err := ranl.Run(); err != nil {
        panic(err)
    }
}

func flipAlpha(s []byte) []byte {
	dt := make([]byte, len(s))
	copy(dt, s)

	for i, r := range s {
		if unicode.IsLetter(rune(r)) || unicode.IsNumber(rune(r)) {
			if r == 'z' {
				dt[i] = 'a'
			} else if r == 'Z' {
				dt[i] = 'A'
			} else {
				dt[i] = r + 1
			}
			break
		}
	}
	return dt
}
