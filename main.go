package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

const (
	separator = "=>"
)

var (
	typeRegex              = regexp.MustCompile(`type:.eq.(\[\d+\])?([a-z].*)`)
	typeSlice              = []byte("type:.eq.")
	mingwFaultySymbol      = []byte("mingw_vgprintf")
	mingwReplacementSymbol = []byte("mingw_vfprintf")
)

type symbl struct {
	symType string
	symVal  []byte
}

func main() {
	archive := os.Args[1]
	nm := os.Args[2]
	ranlib := strings.Join(os.Args[3:], " ")

	buf := new(bytes.Buffer)
	c := exec.Command(nm, archive)
	c.Stdout = buf
	if err := c.Run(); err != nil {
		panic(err)
	}

	symbols := make(map[string][][]byte)
	outch := make(chan *symbl, 100)
	inch := make(chan []byte, 100)
	done := make(chan struct{})

	go func() {
		for s := range outch {
			symbols[s.symType] = append(symbols[s.symType], s.symVal)
		}
		done <- struct{}{}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go parse(inch, outch, &wg)
	}

	content := bytes.Split(buf.Bytes(), []byte{'\n'})
	for _, line := range content {
		splitted := bytes.Split(line, []byte{' '})
		if len(splitted) >= 3 {
			if bytes.ContainsAny(splitted[1], "TSt") {
				inch <- splitted[2]
			}
		}
	}
	close(inch)

	wg.Wait()
	close(outch)
	<-done

	f, err := os.OpenFile(archive, os.O_RDWR, 0755)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	data, _ := io.ReadAll(f)

	for tp, syms := range symbols {
		for _, symbol := range syms {
			switch tp {
			case "simple":
				data = bytes.ReplaceAll(data, symbol, flipAlpha(symbol))
			case "typed":
				toReplace := append(typeSlice, symbol...)
				splitted := bytes.Split(symbol, []byte(separator))
				if len(splitted) == 1 {
					flippedSymbol := append(typeSlice, flipAlpha(symbol)...)
					data = bytes.ReplaceAll(data, toReplace, flippedSymbol)
				} else {
					originalSymbol := append(typeSlice, splitted[0]...)
					originalSymbol = append(originalSymbol, splitted[1]...)
					flipped := flipAlpha(splitted[1])
					flippedSymbol := append(typeSlice, splitted[0]...)
					flippedSymbol = append(flippedSymbol, flipped...)
					data = bytes.ReplaceAll(data, originalSymbol, flippedSymbol)
					data = bytes.ReplaceAll(data, splitted[1], flipped)
				}
			}
		}
	}

	data = bytes.ReplaceAll(data, mingwFaultySymbol, mingwReplacementSymbol)

	f.Truncate(0)
	f.Seek(0, 0)
	f.Write(data)

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

func parse(inch chan []byte, outch chan *symbl, wg *sync.WaitGroup) {
	defer wg.Done()
	for symbol := range inch {
		if !bytes.ContainsAny(symbol, "/.:") && !bytes.Contains(symbol, []byte("frida")) {
			outch <- &symbl{symType: "simple", symVal: symbol}
		}
		if bytes.Contains(symbol, []byte("type:.eq.")) {
			matches := typeRegex.FindSubmatch(symbol)
			if len(matches) > 0 {
				val := bytes.Join(matches[1:], []byte("=>"))
				outch <- &symbl{symType: "typed", symVal: val}
			}
		}
	}
}
