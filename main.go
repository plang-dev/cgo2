package main

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"syscall"
	"unsafe"
)

const maxDepth = 16

type CGOBlock struct {
	pgosp  *uintptr
	pcsp   *uintptr
	gosp   [maxDepth]uintptr
	csp    [maxDepth]uintptr
	p      [maxDepth]*CGOBlock
	pp     int
	g      uintptr
	cblock []byte
	fns    map[string]int
}

func asmjmpgowrite()
func asmgetg() uintptr
func asmsetcgob(p *CGOBlock)
func asmgetcgob() *CGOBlock
func asmcallc(fn, csp uintptr)

func c2goBytes(b, c uintptr) []byte {
	bb := []byte{}
	bbsh := (*reflect.SliceHeader)(unsafe.Pointer(&bb))
	bbsh.Data = b
	bbsh.Cap = int(c)
	bbsh.Len = int(c)
	return bb
}

var fds = []*os.File{
	os.Stdin,
	os.Stdout,
	os.Stderr,
}

func gowrite(fd, p, len uintptr) {
	b := c2goBytes(p, len)
	fds[fd].Write(b)
}

func callc0(gb *CGOBlock, fn int) {
	// g := asmgetg()

	// if gb.pp == 0 {
	// 	gb.g = g
	// } else if gb.g != g {
	// 	panic("call in different goroutine")
	// }

	pgb := asmgetcgob()
	csp := gb.csp[gb.pp]
	gb.pgosp = &gb.gosp[gb.pp]
	gb.p[gb.pp] = pgb
	gb.pp++
	gb.pcsp = &gb.csp[gb.pp]
	asmsetcgob(gb)

	cblock := uintptr(unsafe.Pointer(&gb.cblock[0]))
	asmcallc(cblock+uintptr(fn), csp)

	asmsetcgob(pgb)
	gb.pp--
	gb.p[gb.pp] = nil
}

func createCGOBlock(filename string) (*CGOBlock, error) {
	objfile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	f, err := macho.NewFile(bytes.NewReader(objfile))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type gofunc struct {
		stubOff int
		name    string
		fn      interface{}
	}

	gofuncs := []*gofunc{
		{name: "_write", fn: asmjmpgowrite},
	}
	gonamefuncs := map[string]*gofunc{}
	for _, f := range gofuncs {
		gonamefuncs[f.name] = f
	}

	stubs := []byte{}

	for _, f := range gofuncs {
		f.stubOff = len(stubs)
		b := make([]byte, 128)
		p := 0
		// movabsq $go_func_addr, %eax
		// 48 b8 $go_func_addr(8 bytes)
		// jmp *%eax
		// ff e0
		b[p] = 0x48
		p++
		b[p] = 0xb8
		p++
		binary.LittleEndian.PutUint64(b[p:], uint64(reflect.ValueOf(f.fn).Pointer()))
		p += 8
		b[p] = 0xff
		p++
		b[p] = 0xe0
		p++
		stubs = append(stubs, b[:p]...)
	}

	// fmt.Printf("%+v\n", f.Symtab.Syms)

	// block:
	// stack
	// text
	// stub

	stackBottom := 1024 * 16
	objStart := stackBottom + 1024
	stubStart := objStart + len(objfile)

	fns := map[string]int{}
	for _, s := range f.Symtab.Syms {
		if s.Type == 15 {
			fns[s.Name] = objStart + int(f.Sections[s.Sect-1].Offset) + int(s.Value)
		}
	}

	for _, s := range f.Sections {
		// fmt.Printf("%+v\n", s)
		if s.Name == "__text" {
			for _, r := range s.Relocs {
				// when Scattered == false && Extern == true, Value is the symbol number.
				// when Scattered == false && Extern == false, Value is the section number.
				// when Scattered == true, Value is the value that this reloc refers to.
				if r.Scattered {
					panic("unsupported")
				}
				if r.Extern {
					if r.Type != uint8(macho.X86_64_RELOC_BRANCH) {
						panic("unsupported")
					}
					if r.Len != 2 {
						panic("unsupported")
					}
					name := f.Symtab.Syms[r.Value].Name
					f, ok := gonamefuncs[name]
					if !ok {
						return nil, fmt.Errorf("%s not supported", name)
					}
					p := objStart + int(s.Offset) + int(r.Addr)
					dst := stubStart + f.stubOff
					delta := dst - (p + 4)
					binary.LittleEndian.PutUint32(objfile[p-objStart:], uint32(delta))
				}
			}
		}
	}

	cblockSize := 1024 * 128
	cblock, err := syscall.Mmap(
		0,
		0,
		cblockSize,
		syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC,
		syscall.MAP_PRIVATE|syscall.MAP_ANON,
	)
	if err != nil {
		return nil, err
	}

	for i := 0; i < cblockSize; i++ {
		cblock[i] = 0
	}
	copy(cblock[objStart:], objfile)
	copy(cblock[stubStart:], stubs)

	gb := &CGOBlock{
		cblock: cblock,
		fns:    fns,
	}
	gb.csp[0] = uintptr(unsafe.Pointer(&cblock[0])) + uintptr(stackBottom)

	return gb, nil
}

func run() error {
	flag.Parse()
	args := flag.Args()
	filename := args[0]
	fn := args[1]

	gb, err := createCGOBlock(filename)
	if err != nil {
		return err
	}

	callc0(gb, gb.fns[fn])

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
