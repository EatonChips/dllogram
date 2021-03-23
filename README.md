# dllogram

Generates cpp code that can be compiled into an exe or dll that executes a specified shellcode. Can parse an existing dll and implement functions by proxying to retain functionality. 

## Usage

### Examples

This will generate an exe which executes the provided shellcode.

```
dllogram -i shellcode.bin -f exe -build
```

This will generate a dll which executes the provided shellcode when attached to.

```
dllogram -i shellcode.bin -f dll -build
```

This will generate a dll which proxies the original functionality of `msvcp140.dll`. 

```
dllogram -i shellcode.bin -f dll -proxy-dll msvcp140.dll -build
```

```
dllogram 
	-a int
        Architecture: 32, 64 (default 64)
  -build
        Build generated code?
  -f string
        Executable format: dll, exe (default "exe")
  -i string
        Shellcode file
  -o string
        Output file
  -proxy-dll string
        DLL to proxy functions to
```

#### -a

Shellcode/target system architecture, 32/64 bit.

#### -build

Attempts to build the generated code using mingw commands below.

#### -f

Format of payload, currently supported options are an `exe` or `dll`.

#### -i

Raw shellcode input file.

#### -o

Compiler output file if build is specified. Is automatically set to dll name when using `-dll-proxy` option.

#### -proxy-dll

DLL to proxy functions to. This will rename the target dll with a random extension and place it in the build directory. It will also set the output file name to that of the original target dll.

### Building with mingw:

#### 64 bit

```
x86_64-w64-mingw32-g++ -o nice.exe build/*

x86_64-w64-mingw32-g++ -shared -o msvcp140.dll build/*
```

#### 32 bit

```
i686-w64-mingw32-g++ -o nice.exe build/*

i686-w64-mingw32-g++ -shared -o msvcp140.dll build/*
```

## Future

This is designed to be flexible using Go's templating engine. In the future I may add functionality to generate payloads which implement different methods to execute or obfuscate shellcode.

## Thanks...

- to [@S4R1N](https://github.com/S4R1N) for showing me the power of DLL proxying and explaining windows internals

